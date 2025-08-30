package postgres

import(
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/CodenSell/WB_test_level0/internal/structs"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(user, password, dbname, host string, port int) (*Repository, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

func (r *Repository) GetOrder(orderUID string) (*structs.Order, error) {
	var o structs.Order

	err := r.db.QueryRow(`
		SELECT order_uid, track_number, entry, locale, internal_signature,
		       customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid=$1
	`, orderUID).Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Localization, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.StorageID,
		&o.DateCreated, &o.OofShard,
	)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid=$1
	`, orderUID).Scan(
		&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.ZIP, &o.Delivery.City,
		&o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,
	)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank,
		       delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid=$1
	`, orderUID).Scan(
		&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency,
		&o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDT,
		&o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal,
		&o.Payment.CustomFee,
	)
	if err != nil {
		return nil, err
	}
 
	rows, err := r.db.Query(`
		SELECT chrt_id, track_number, price, rid, name, sale, size,
		       total_price, nm_id, brand, status
		FROM items WHERE order_uid=$1
	`, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it structs.Items
		if err := rows.Scan(
			&it.ChartID, &it.TrackNumber, &it.Price, &it.Rid,
			&it.Name, &it.Sale, &it.Size, &it.TotalPrice,
			&it.NomenclatureID, &it.Brand, &it.Status,
		); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}

	return &o, nil
}