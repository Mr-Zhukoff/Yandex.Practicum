package events

import "fmt"

func ValidateProduct(product Product) error {
	if product.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	if product.Name == "" {
		return fmt.Errorf("name is required")
	}
	if product.Price.Amount < 0 {
		return fmt.Errorf("price.amount must be non-negative")
	}
	if product.Price.Currency == "" {
		return fmt.Errorf("price.currency is required")
	}
	if product.Stock.Available < 0 || product.Stock.Reserved < 0 {
		return fmt.Errorf("stock values must be non-negative")
	}
	return nil
}
