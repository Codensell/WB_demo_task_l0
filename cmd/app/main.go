package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/CodenSell/WB_test_level0/internal/api"
	"github.com/CodenSell/WB_test_level0/internal/cache"
	"github.com/CodenSell/WB_test_level0/internal/storage/postgres"
	"github.com/CodenSell/WB_test_level0/internal/structs"
	"github.com/segmentio/kafka-go"
)

type App struct {
	cache *cache.Cache
	repo  *postgres.Repository
}

func (a *App) consumeKafka(ctx context.Context, brokers []string, topic, group string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  group,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(ctx)
		log.Println(m)
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

		a.cache.CreateOrder(ctx, &o)

		log.Printf("order %s saved from kafka", o.OrderUID)
	}
}

func main() {
	repo, err := postgres.NewRepository(
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		"postgres", 5432,
	)
	if err != nil {
		log.Fatal("cant connect to DB:", err)
	}
	log.Println("DB connection true")

	cache := cache.NewCache(repo, "data/model.json")

	tmplIndex := template.Must(template.ParseFiles("internal/templates/index.html"))
	tmplView := template.Must(template.ParseFiles("internal/templates/view.html"))

	handler := api.NewOrderHandler(tmplIndex, tmplView, cache, repo)

	app := &App{cache: cache, repo: repo}

	ctx := context.Background()
	go app.consumeKafka(ctx,
		[]string{os.Getenv("KAFKA_URL")},
		"orders",
		"order-service",
	)

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      handler.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server listens: 8081")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
