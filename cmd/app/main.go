package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/CodenSell/WB_test_level0/internal/api"
	"github.com/CodenSell/WB_test_level0/internal/cache"
	"github.com/CodenSell/WB_test_level0/internal/broker"
	"github.com/CodenSell/WB_test_level0/internal/storage/postgres"
)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := consumer.NewReader(
		consumer.Config{
			Brokers: []string{os.Getenv("KAFKA_URL")},
			Topic:   "orders",
			GroupID: "order-service",
		},
		repo,
		cache,
	)
	go reader.Start(ctx)

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
