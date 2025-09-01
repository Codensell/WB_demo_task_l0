package postgres

import(
	"context"
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
func (r *Repository) UpsertOrder(ctx context.Context, o *structs.Order) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature,
		                    customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET
		    track_number=EXCLUDED.track_number,
		    entry=EXCLUDED.entry,
		    locale=EXCLUDED.locale,
		    internal_signature=EXCLUDED.internal_signature,
		    customer_id=EXCLUDED.customer_id,
		    delivery_service=EXCLUDED.delivery_service,
		    shardkey=EXCLUDED.shardkey,
		    sm_id=EXCLUDED.sm_id,
		    date_created=EXCLUDED.date_created,
		    oof_shard=EXCLUDED.oof_shard
	`, o.OrderUID, o.TrackNumber, o.Entry, o.Localization, o.InternalSignature,
		o.CustomerID, o.DeliveryService, o.ShardKey, o.StorageID, o.DateCreated, o.OofShard)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (order_uid) DO UPDATE SET
		    name=EXCLUDED.name,
		    phone=EXCLUDED.phone,
		    zip=EXCLUDED.zip,
		    city=EXCLUDED.city,
		    address=EXCLUDED.address,
		    region=EXCLUDED.region,
		    email=EXCLUDED.email
	`, o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.ZIP, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount,
		                      payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET
		    transaction=EXCLUDED.transaction,
		    request_id=EXCLUDED.request_id,
		    currency=EXCLUDED.currency,
		    provider=EXCLUDED.provider,
		    amount=EXCLUDED.amount,
		    payment_dt=EXCLUDED.payment_dt,
		    bank=EXCLUDED.bank,
		    delivery_cost=EXCLUDED.delivery_cost,
		    goods_total=EXCLUDED.goods_total,
		    custom_fee=EXCLUDED.custom_fee
	`, o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDT, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderUID); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size,
		                   total_price, nm_id, brand, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, it := range o.Items {
		if _, err = stmt.ExecContext(ctx, o.OrderUID, it.ChartID, it.TrackNumber, it.Price, it.Rid,
			it.Name, it.Sale, it.Size, it.TotalPrice, it.NomenclatureID, it.Brand, it.Status); err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}
func (r *Repository)ListOrderUIDs(ctx context.Context)([]string, error){
	rows, err := r.db.QueryContext(ctx, `SELECT order_uid FROM orders`)
	if err != nil{
		return nil, err
	}
	defer rows.Close()
	var uids []string
	for rows.Next(){
		var uid string
		if err := rows.Scan(&uid); err != nil{
			return nil, err
		}
		uids = append(uids, uid)
	}
	return uids, rows.Err()
}