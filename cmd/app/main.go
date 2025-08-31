package main

import (
	"encoding/json"
	"context"
	"os"
	"sync"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
	"github.com/CodenSell/WB_test_level0/internal/structs"
	"github.com/CodenSell/WB_test_level0/internal/storage/postgres"
	"github.com/segmentio/kafka-go"
)

type App struct {
	tmplIndex *template.Template
	tmplView  *template.Template
	mu sync.RWMutex
	cache map[string]structs.Order
	repo *postgres.Repository
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/view", a.handleView)
	mux.HandleFunc("/order/", a.handleAPI)
	return mux
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.tmplIndex.Execute(w, nil)
}

func (a *App) handleView(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(r.URL.Query().Get("order_uid"))
	if uid == "" {
		http.Error(w, "need order_uid", http.StatusBadRequest)
		return
	}
	a.mu.RLock()
	order, ok := a.cache[uid]
	a.mu.RUnlock()
	if !ok{
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.tmplView.Execute(w, order)
}

func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimPrefix(r.URL.Path, "/order/")
	if uid == "" || strings.Contains(uid, "/") {
		http.Error(w, "were waiting for path /order/{order_uid}", http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	order, ok := a.cache[uid]
	a.mu.RUnlock()

	if !ok {
		fromDB, err := a.repo.GetOrder(uid)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		order = *fromDB
		a.mu.Lock()
		a.cache[uid] = order
		a.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(order)
}
func (a *App) consumeKafka(ctx context.Context, brokers []string, topic, group string) {
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers: brokers,
        GroupID: group,
        Topic: topic,
        MinBytes: 1,
        MaxBytes: 10e6,
    })
    defer r.Close()

    for {
        m, err := r.ReadMessage(ctx)
        if err != nil {
            log.Printf("kafka read error: %v", err)
            return
        }

        var o structs.Order
        if err := json.Unmarshal(m.Value, &o); err != nil {
            log.Printf("cant unmarshal message: %v", err)
            continue
        }
        if o.OrderUID == "" {
            log.Printf("invalid order, empty uid")
            continue
        }

        if err := a.repo.UpsertOrder(ctx, &o); err != nil {
            log.Printf("db upsert error: %v", err)
            continue
        }
        a.mu.Lock()
        a.cache[o.OrderUID] = o
        a.mu.Unlock()

        log.Printf("order %s saved from kafka", o.OrderUID)
    }
}

func (a *App) readAndLoadFromFile(path string){
	data, err := os.ReadFile(path)
	if err != nil{
		log.Printf("cant read %s: %v", path, err)
		return
	}
	var o structs.Order
	if err := json.Unmarshal(data, &o); err != nil{
		log.Printf("cant unmarshal: %v", err)
		return
	}
	if o.OrderUID == ""{
		log.Printf("empty order_uid")
		return
	}
	a.mu.Lock()
	a.cache[o.OrderUID] = o
	a.mu.Unlock()
	log.Printf("loaded order %s", o.OrderUID)
}

func main() {
	repo, err := postgres.NewRepository(
		"tester", "123", "wbrrs", "localhost", 5432,
	)
	if err != nil{
		log.Fatal("cant connect to DB:", err)
	}
	log.Println("DB connection true")
	tmplIndex := template.Must(template.ParseFiles("internal/templates/index.html"))
	tmplView := template.Must(template.ParseFiles("internal/templates/view.html"))

	app := &App{tmplIndex: tmplIndex, tmplView: tmplView, cache: make(map[string]structs.Order), repo: repo}

	app.readAndLoadFromFile("data/model.json")
	ctx := context.Background()
	go app.consumeKafka(ctx,
		[]string{"localhost:9092"},
		"orders",
		"order-service",
	)

	srv := &http.Server{
		Addr: ":8081",
		Handler: app.routes(),
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	log.Println("Server listens: 8081")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
