package main

import (
	"html/template" //чтобы обойтись без .js при создании веб интерфейса для задачи Л0
	"log"
	"net/http" //стандартная библиотека Го, отвечает за сервер, маршрутизацию и готовые интерфейсы
	"strings"
	"time"
)

//Попробую проверить работоспособность веб интерфейса index.html/view.html

type App struct {
	tmplIndex *template.Template
	tmplView  *template.Template
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
func (a *App) handleView(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(r.URL.Query().Get("order_uid"))
	if uid == "" {
		http.Error(w, "need order_uid", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct{ UID string }{UID: uid}
	_ = a.tmplView.Execute(w, data)
}

// обработчик http запроса, который отдаст json в ответ
func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	log.Printf("API hit: %s", r.URL.Path)            // для диагностики
	uid := strings.TrimPrefix(r.URL.Path, "/order/") // убирает путь до номера заказа
	if uid == "" || strings.Contains(uid, "/") {
		http.Error(w, "were waiting for path /order/{order_uid}", http.StatusBadRequest)
		return
	}
	// сейчас просто заглушка, для проверки интерфейса
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not found"}`))

}

func main() {
	//функция Must() находится в пакете html/templates, которая принимает результат парсинга без проверок
	//функция ParseFiles() в том же пакете, загружает html файлы и возвращает ошибку и указатель на Template
	tmplIndex := template.Must(template.ParseFiles("internal/templates/index.html"))
	tmplView := template.Must(template.ParseFiles("internal/templates/view.html"))

	// присваиваю структуру App переменной, чтобы потом передать в обработчик (handler)
	app := &App{tmplIndex: tmplIndex, tmplView: tmplView}

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
