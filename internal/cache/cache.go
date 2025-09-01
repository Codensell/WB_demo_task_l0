package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"database/sql"

	"github.com/CodenSell/WB_test_level0/internal/storage/postgres"
	"github.com/CodenSell/WB_test_level0/internal/structs"
)

type Cache struct {
	mu    sync.RWMutex
	cache map[string]structs.Order
	repo  *postgres.Repository
}

func NewCache(repo *postgres.Repository, path string) *Cache {
	cache := &Cache{
		repo:  repo,
		cache: make(map[string]structs.Order),
	}
	cache.readAndLoadFromFile(path)

	ctx := context.Background()
	cache.fillUpCache(ctx)

	return cache
}

func (a *Cache) fillUpCache(ctx context.Context) {
	uids, err := a.repo.ListOrderUIDs(ctx)
	if err != nil {
		log.Printf("cache preload error: %v", err)
		return
	}
	count := 0
	for _, uid := range uids {
		o, err := a.repo.GetOrder(uid)
		if err != nil {
			log.Printf("preload get %s: %v", uid, err)
			continue
		}
		a.mu.Lock()
		a.cache[o.OrderUID] = *o
		a.mu.Unlock()
		count++
	}
	log.Printf("cache preload done, loaded %d orders", count)
}

func (a *Cache) readAndLoadFromFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("cant read %s: %v", path, err)
		return
	}
	var o structs.Order
	if err := json.Unmarshal(data, &o); err != nil {
		log.Printf("cant unmarshal: %v", err)
		return
	}
	if o.OrderUID == "" {
		log.Printf("empty order_uid")
		return
	}
	a.mu.Lock()
	a.cache[o.OrderUID] = o
	a.mu.Unlock()
	log.Printf("loaded order %s", o.OrderUID)
}

func (a *Cache) CreateOrder(ctx context.Context, o *structs.Order) error {
	if o == nil {
		return errors.New("nil order")
	}
	uid := strings.TrimSpace(o.OrderUID)
	if uid == "" {
		return errors.New("empty order_uid")
	}
	if err := a.repo.UpsertOrder(ctx, o); err != nil {
		return err
	}
	a.mu.Lock()
	a.cache[uid] = *o
	a.mu.Unlock()
	return nil
}

func (a *Cache) GetOrder(ctx context.Context, uid string) (*structs.Order, bool, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, false, errors.New("empty uid")
	}

	a.mu.RLock()
	if o, ok := a.cache[uid]; ok {
		a.mu.RUnlock()
		return &o, true, nil
	}
	a.mu.RUnlock()

	o, err := a.repo.GetOrder(uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	a.mu.Lock()
	a.cache[uid] = *o
	a.mu.Unlock()

	return o, true, nil
}
