package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/CodenSell/WB_test_level0/internal/cache"
	"github.com/CodenSell/WB_test_level0/internal/storage"
	"github.com/CodenSell/WB_test_level0/internal/structs"
	"github.com/CodenSell/WB_test_level0/internal/validation"
)

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

type Reader struct {
	cfg   Config
	repo  storage.OrderRepo
	cache *cache.Cache
	r     *kafka.Reader
}

func NewReader(cfg Config, repo storage.OrderRepo, cache *cache.Cache) *Reader {
	return &Reader{
		cfg:   cfg,
		repo:  repo,
		cache: cache,
		r: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  cfg.Brokers,
			GroupID:  cfg.GroupID,
			Topic:    cfg.Topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
	}
}

func (c *Reader) Start(ctx context.Context) {
	defer c.r.Close()

	backoff := time.Second

	for {
		m, err := c.r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka fetch error: %v", err)
			select {
			case <-time.After(backoff):
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			case <-ctx.Done():
				return
			}
		}
		backoff = time.Second

		var o structs.Order
		if err := json.Unmarshal(m.Value, &o); err != nil {
			log.Printf("cant unmarshal message: %v", err)
			_ = c.r.CommitMessages(ctx, m)
			continue
		}
		if err := validation.ValidateOrder(&o); err != nil {
			log.Printf("skip invalid order: %v", err)
			_ = c.r.CommitMessages(ctx, m)
			continue
		}

		if err := c.repo.UpsertOrder(ctx, &o); err != nil {
			log.Printf("db upsert error: %v", err)
			continue
		}

		if err := c.cache.CreateOrder(ctx, &o); err != nil {
			log.Printf("cache create error: %v", err)
		}

		if err := c.r.CommitMessages(ctx, m); err != nil {
			log.Printf("commit error: %v", err)
		}

		log.Printf("order %s saved and committed from kafka", o.OrderUID)
	}
}
