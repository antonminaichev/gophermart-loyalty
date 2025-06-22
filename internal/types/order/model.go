package order

import "time"

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusProcessed  OrderStatus = "PROCESSED"
	StatusInvalid    OrderStatus = "INVALID"
)

type Order struct {
	ID            int64       `db:"id" json:"-"`
	UserID        int64       `db:"user_id" json:"-"`
	Number        string      `db:"number" json:"number"`
	Status        OrderStatus `db:"status" json:"status"`
	AccrualAmount *float64    `db:"accrual_amount" json:"accrual,omitempty"`
	UploadedAt    time.Time   `db:"uploaded_at" json:"uploaded_at"`
	ProcessedAt   *time.Time  `db:"processed_at" json:"-"`
}
