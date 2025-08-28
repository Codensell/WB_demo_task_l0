package main

import (
	"encoding/json"
	"os"
	"sync"
	"html/template" //чтобы обойтись без .js при создании веб интерфейса для задачи Л0
	"log"
	"net/http" //стандартная библиотека Го, отвечает за сервер, маршрутизацию и готовые интерфейсы
	"strings"
	"time"
	"github.com/CodenSell/WB_test_level0/internal/structs"
)

//Попробую проверить работоспособность веб интерфейса index.html/view.html

type App struct {
	tmplIndex *template.Template
	tmplView  *template.Template
	// дополняем мапой с ключом order_uid и значением структуры Order + mutex для защиты кеша
	mu sync.RWMutex
	cache map[string]structs.Order
}

// Маршрутизация
func (a *App) routes() http.Handler {
	mux := http.NewServeMux() // создаю объект роутера и сохраняем в переменной
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/view", a.handleView)
	mux.HandleFunc("/order/", a.handleAPI) // тут наш json
	return mux
}

// обработчик index.html
func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.tmplIndex.Execute(w, nil)
}

// обработчик view.html
// добавил мьютексы для безопасного параллельного чтения и записи кеша
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

// обработчик http запроса, который отдаст json в ответ
//
func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimPrefix(r.URL.Path, "/order/") // убирает путь до номера заказа
	if uid == "" || strings.Contains(uid, "/") {
		http.Error(w, "were waiting for path /order/{order_uid}", http.StatusBadRequest)
		return
	}
	//читаем из кеша
	a.mu.RLock()
	order, ok := a.cache[uid]
	a.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !ok{
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
		return
	}
	//если ошибок нет то преобразуем go структуру в json
	enc := json.NewEncoder(w)
	_ = enc.Encode(order)

}
//функция для чтения файла, преобразования json в go struct и сохраняю в кеш
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
	//функция Must() находится в пакете html/templates, которая принимает результат парсинга без проверок
	//функция ParseFiles() в том же пакете, загружает html файлы и возвращает ошибку и указатель на Template
	tmplIndex := template.Must(template.ParseFiles("internal/templates/index.html"))
	tmplView := template.Must(template.ParseFiles("internal/templates/view.html"))

	// присваиваю структуру App переменной, чтобы потом передать в обработчик (handler)
	app := &App{tmplIndex: tmplIndex, tmplView: tmplView, cache: make(map[string]structs.Order)}

	app.readAndLoadFromFile("model.json")

	srv := &http.Server{
		Addr:    ":8081",
		Handler: app.routes(),
		// в данном случае timeout не особо нужны
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server listens: 8081")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
