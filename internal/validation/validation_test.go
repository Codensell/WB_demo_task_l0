package validation

import (
	"testing"

	"github.com/CodenSell/WB_test_level0/internal/structs"
)

func TestValidateOrder_OK(t *testing.T) {
	o := &structs.Order{
		OrderUID: "ok",
		Delivery: structs.Delivery{
			Email: "a@b.c", Phone: "123",
		},
		Payment: structs.Payment{
			Amount: 100, DeliveryCost: 10, GoodsTotal: 90,
		},
		Items: []structs.Items{
			{Name: "item", Price: 10, TotalPrice: 10},
		},
	}
	if err := ValidateOrder(o); err != nil {
		t.Fatalf("expected ok, got err: %v", err)
	}
}

func TestValidateOrder_EmptyUID(t *testing.T) {
	o := &structs.Order{
		OrderUID: "",
		Delivery: structs.Delivery{Email: "a@b.c", Phone: "1"},
		Payment:  structs.Payment{Amount: 1},
		Items:    []structs.Items{{Name: "x", Price: 1, TotalPrice: 1}},
	}
	if err := ValidateOrder(o); err == nil {
		t.Fatalf("expected error for empty uid")
	}
}

func TestValidateOrder_NoContacts(t *testing.T) {
	o := &structs.Order{
		OrderUID: "x",
		Delivery: structs.Delivery{Email: "", Phone: ""},
		Payment:  structs.Payment{Amount: 1},
		Items:    []structs.Items{{Name: "x", Price: 1, TotalPrice: 1}},
	}
	if err := ValidateOrder(o); err == nil {
		t.Fatalf("expected error for no contacts")
	}
}

func TestValidateOrder_NegativeNumbers(t *testing.T) {
	o := &structs.Order{
		OrderUID: "x",
		Delivery: structs.Delivery{Email: "a@b.c", Phone: "1"},
		Payment:  structs.Payment{Amount: -1, DeliveryCost: 0, GoodsTotal: 0},
		Items:    []structs.Items{{Name: "x", Price: 1, TotalPrice: 1}},
	}
	if err := ValidateOrder(o); err == nil {
		t.Fatalf("expected error for negative amount")
	}
}

func TestValidateOrder_EmptyItems(t *testing.T) {
	o := &structs.Order{
		OrderUID: "x",
		Delivery: structs.Delivery{Email: "a@b.c", Phone: "1"},
		Payment:  structs.Payment{Amount: 1, DeliveryCost: 0, GoodsTotal: 1},
		Items:    []structs.Items{},
	}
	if err := ValidateOrder(o); err == nil {
		t.Fatalf("expected error for empty items")
	}
}

func TestValidateOrder_ItemHasEmptyName(t *testing.T) {
	o := &structs.Order{
		OrderUID: "x",
		Delivery: structs.Delivery{Email: "a@b.c", Phone: "1"},
		Payment:  structs.Payment{Amount: 1, DeliveryCost: 0, GoodsTotal: 1},
		Items:    []structs.Items{{Name: "", Price: 1, TotalPrice: 1}},
	}
	if err := ValidateOrder(o); err == nil {
		t.Fatalf("expected error for empty item name")
	}
}
