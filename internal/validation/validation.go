package validation

import (
	"errors"
	"strconv"
	"strings"

	"github.com/CodenSell/WB_test_level0/internal/structs"
)

func ValidateOrder(o *structs.Order) error {
	if o == nil {
		return errors.New("nil order")
	}
	if strings.TrimSpace(o.OrderUID) == "" {
		return errors.New("invalid value: UID empty")
	}
	if o.Delivery.Email == "" || o.Delivery.Phone == "" {
		return errors.New("invalid valus: no email or phone number")
	}
	if o.Payment.Amount < 0 || o.Payment.DeliveryCost < 0 || o.Payment.GoodsTotal < 0 {
		return errors.New("invalid value: negative values")
	}
	if len(o.Items) == 0 {
		return errors.New("invalid value: no items in list")
	}
	for i, it := range o.Items {
		if it.Price < 0 || it.TotalPrice < 0 {
			return errors.New("invalid valeu: negative price" + strconv.Itoa(i))
		}
		if strings.TrimSpace(it.Name) == "" {
			return errors.New("invalid value: empty name of goods" + strconv.Itoa(i))
		}
	}
	return nil
}
