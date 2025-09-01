package storage

import(
	"context"
	"github.com/CodenSell/WB_test_level0/internal/structs"
)

type OrderRepo interface{
	GetOrder(ctx context.Context, uid string)(*structs.Order, error)
	UpsertOrder(ctx context.Context, o *structs.Order) error
	ListOrderUIDs(ctx context.Context)([]string, error)
}