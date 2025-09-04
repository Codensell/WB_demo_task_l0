package postgres

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/CodenSell/WB_test_level0/internal/structs"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func mustRepo(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	r := &Repository{db: db}
	cleanup := func() { _ = db.Close() }
	return r, mock, cleanup
}

func TestGetOrder_OK(t *testing.T) {
	r, mock, done := mustRepo(t)
	defer done()

	orderUID := "uid-1"

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT order_uid, track_number, entry, locale, internal_signature,
		       customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid=$1`)).
		WithArgs(orderUID).
		WillReturnRows(sqlmock.NewRows([]string{
			"order_uid", "track_number", "entry", "locale", "internal_signature",
			"customer_id", "delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
		}).AddRow(orderUID, "WBTR", "WBIL", "en", "",
			"cust", "meest", "9", 99, time.Now().UTC().Format(time.RFC3339), "1"))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid=$1`)).
		WithArgs(orderUID).
		WillReturnRows(sqlmock.NewRows([]string{
			"name", "phone", "zip", "city", "address", "region", "email",
		}).AddRow("Name", "+1", "000", "City", "Addr", "Reg", "a@b.c"))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank,
		       delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid=$1`)).
		WithArgs(orderUID).
		WillReturnRows(sqlmock.NewRows([]string{
			"transaction", "request_id", "currency", "provider", "amount", "payment_dt", "bank",
			"delivery_cost", "goods_total", "custom_fee",
		}).AddRow("tx", "", "USD", "wbpay", 100, int64(1637907727), "alpha", 10, 90, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT chrt_id, track_number, price, rid, name, sale, size,
		       total_price, nm_id, brand, status
		FROM items WHERE order_uid=$1`)).
		WithArgs(orderUID).
		WillReturnRows(sqlmock.NewRows([]string{
			"chrt_id", "track_number", "price", "rid", "name", "sale", "size",
			"total_price", "nm_id", "brand", "status",
		}).AddRow(1, "WBTR", 10, "rid", "Mask", 0, "0", 10, 111, "Brand", 202))

	o, err := r.GetOrder(context.Background(), orderUID)
	if err != nil {
		t.Fatalf("GetOrder err: %v", err)
	}
	if o.OrderUID != orderUID || len(o.Items) != 1 || o.Payment.Amount != 100 {
		t.Fatalf("bad aggregate: %+v", o)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertOrder_OK(t *testing.T) {
	r, mock, done := mustRepo(t)
	defer done()

	o := &structs.Order{
		OrderUID:        "u1",
		TrackNumber:     "WBTR",
		Entry:           "WBIL",
		Localization:    "en",
		CustomerID:      "cust",
		DeliveryService: "meest",
		ShardKey:        "9",
		StorageID:       99,
		DateCreated:     time.Now().UTC().Format(time.RFC3339),
		OofShard:        "1",
		Delivery: structs.Delivery{
			Name: "N", Phone: "+1", ZIP: "000", City: "C", Address: "A", Region: "R", Email: "a@b.c",
		},
		Payment: structs.Payment{
			Transaction: "tx", RequestID: "", Currency: "USD", Provider: "wbpay",
			Amount: 100, PaymentDT: 1637907727, Bank: "alpha", DeliveryCost: 10, GoodsTotal: 90, CustomFee: 0,
		},
		Items: []structs.Items{
			{ChartID: 1, TrackNumber: "WBTR", Price: 10, Rid: "rid", Name: "Mask", Sale: 0, Size: "0", TotalPrice: 10, NomenclatureID: 111, Brand: "Brand", Status: 202},
		},
	}

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`
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
		    oof_shard=EXCLUDED.oof_shard`,
	)).WithArgs(
		o.OrderUID, o.TrackNumber, o.Entry, o.Localization, o.InternalSignature,
		o.CustomerID, o.DeliveryService, o.ShardKey, o.StorageID, o.DateCreated, o.OofShard,
	).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (order_uid) DO UPDATE SET
		    name=EXCLUDED.name,
		    phone=EXCLUDED.phone,
		    zip=EXCLUDED.zip,
		    city=EXCLUDED.city,
		    address=EXCLUDED.address,
		    region=EXCLUDED.region,
		    email=EXCLUDED.email`,
	)).WithArgs(
		o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.ZIP, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email,
	).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
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
		    custom_fee=EXCLUDED.custom_fee`,
	)).WithArgs(
		o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDT, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee,
	).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM items WHERE order_uid=$1`)).
		WithArgs(o.OrderUID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectPrepare(regexp.QuoteMeta(`
		INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size,
		                   total_price, nm_id, brand, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`)).
		ExpectExec().
		WithArgs(o.OrderUID, o.Items[0].ChartID, o.Items[0].TrackNumber, o.Items[0].Price, o.Items[0].Rid,
			o.Items[0].Name, o.Items[0].Sale, o.Items[0].Size, o.Items[0].TotalPrice, o.Items[0].NomenclatureID, o.Items[0].Brand, o.Items[0].Status).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := r.UpsertOrder(context.Background(), o); err != nil {
		t.Fatalf("UpsertOrder err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
