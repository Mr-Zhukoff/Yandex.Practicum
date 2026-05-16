package productio

import (
	"encoding/json"
	"fmt"
	"os"

	"marketplace-analytics/internal/events"
)

func ReadProducts(path string) ([]events.Product, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var products []events.Product
	if err := json.Unmarshal(data, &products); err == nil {
		return products, nil
	}

	var product events.Product
	if err := json.Unmarshal(data, &product); err == nil {
		return []events.Product{product}, nil
	}

	return nil, fmt.Errorf("%s must contain either a product object or an array of product objects", path)
}
