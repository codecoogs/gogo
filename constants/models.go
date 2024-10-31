package constants

type PurchaseEvent struct {
	UserEmail     string  `json:"user_email"`
	UserName      string  `json:"user_name"`
	ProductName   string  `json:"product_name"`
	ProductID     string  `json:"product_id"`
	PurchasePrice float64 `json:"purchase_price"`
	PurchaseDate  string  `json:"purchase_date"`
}
