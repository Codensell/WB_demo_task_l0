package api

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/CodenSell/WB_test_level0/internal/cache"
	"github.com/CodenSell/WB_test_level0/internal/storage/postgres"
)

type OrderHandler struct {
	tmplIndex *template.Template
	tmplView  *template.Template
	cache     *cache.Cache
	repo      *postgres.Repository
}

func NewOrderHandler(tmplIndex *template.Template, tmplView *template.Template, cache *cache.Cache, repo *postgres.Repository) *OrderHandler {
	return &OrderHandler{tmplIndex: tmplIndex, tmplView: tmplView, cache: cache, repo: repo}
}

func (a *OrderHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/view", a.handleView)
	mux.HandleFunc("/order/", a.handleAPI)
	return mux
}

func (a *OrderHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.tmplIndex.Execute(w, nil)
}

func (a *OrderHandler) handleView(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(r.URL.Query().Get("order_uid"))
	if uid == "" {
		http.Error(w, "need order_uid", http.StatusBadRequest)
		return
	}

	order, found, err := a.cache.GetOrder(r.Context(), uid)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.tmplView.Execute(w, order)
}

func (a *OrderHandler) handleAPI(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimPrefix(r.URL.Path, "/order/")
	if uid == "" || strings.Contains(uid, "/") {
		http.Error(w, "were waiting for path /order/{order_uid}", http.StatusBadRequest)
		return
	}

	order, found, err := a.cache.GetOrder(r.Context(), uid)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}`))
		return
	}
	if !found {
		// не нашли ни в кеше, ни в БД
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(order)
}
