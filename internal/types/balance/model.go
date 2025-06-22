package balance

import "time"

type Withdrawal struct {
	ID          int64     `db:"id" json:"-"`
	UserID      int64     `db:"user_id" json:"-"`
	OrderNumber string    `db:"order_number" json:"order"`
	Amount      float64   `db:"amount" json:"sum"`
	CreatedAt   time.Time `db:"created_at" json:"processed_at"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type BalanceDTO struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
