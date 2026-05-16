package events

import "time"

type EventEnvelope[T any] struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	EventTime time.Time `json:"event_time"`
	Source    string    `json:"source"`
	Payload   T         `json:"payload"`
}

type Product struct {
	ProductID      string                 `json:"product_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Price          Price                  `json:"price"`
	Category       string                 `json:"category"`
	Brand          string                 `json:"brand"`
	Stock          Stock                  `json:"stock"`
	SKU            string                 `json:"sku"`
	Tags           []string               `json:"tags"`
	Images         []Image                `json:"images"`
	Specifications map[string]interface{} `json:"specifications"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Index          string                 `json:"index"`
	StoreID        string                 `json:"store_id"`
}

type Price struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Stock struct {
	Available int `json:"available"`
	Reserved  int `json:"reserved"`
}

type Image struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type SearchRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
}

type RecommendationRequest struct {
	UserID   string `json:"user_id"`
	Category string `json:"category"`
}

type Recommendation struct {
	RecommendationID string               `json:"recommendation_id"`
	UserID           string               `json:"user_id,omitempty"`
	Category         string               `json:"category"`
	Products         []RecommendedProduct `json:"products"`
	CalculatedAt     time.Time            `json:"calculated_at"`
}

type RecommendedProduct struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
}

type ForbiddenProduct struct {
	ProductID string    `json:"product_id"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Active    bool      `json:"active"`
}

type RejectedProduct struct {
	Product Product `json:"product"`
	Reason  string  `json:"reason"`
}

type DeadLetter struct {
	RawPayload string `json:"raw_payload"`
	Reason     string `json:"reason"`
	Source     string `json:"source"`
}
