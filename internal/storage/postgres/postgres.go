package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/antonminaichev/gophermart-loyalty/internal/types/balance"
	"github.com/antonminaichev/gophermart-loyalty/internal/types/order"
	"github.com/antonminaichev/gophermart-loyalty/internal/types/user"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &PostgresStorage{db: db}

	// проверяем, что БД жива
	if err := s.db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	// создаём таблицы
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *PostgresStorage) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            login TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS orders (
            id SERIAL PRIMARY KEY,
            user_id INT NOT NULL REFERENCES users(id),
            number TEXT UNIQUE NOT NULL,
            status TEXT NOT NULL,
            accrual_amount DOUBLE PRECISION,
            uploaded_at TIMESTAMPTZ NOT NULL,
            processed_at TIMESTAMPTZ
        )`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
            id SERIAL PRIMARY KEY,
            user_id INT NOT NULL REFERENCES users(id),
			order_number TEXT NOT NULL,
            amount DOUBLE PRECISION NOT NULL,
            created_at TIMESTAMPTZ NOT NULL
        )`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

func (s *PostgresStorage) Create(ctx context.Context, u *user.User) error {
	q := `INSERT INTO users (login,password_hash,created_at) VALUES($1,$2,$3) RETURNING id`
	return s.db.QueryRowContext(ctx, q, u.Login, u.PasswordHash, u.CreatedAt).Scan(&u.ID)
}

func (s *PostgresStorage) FindByLogin(ctx context.Context, login string) (*user.User, error) {
	u := &user.User{}
	q := `SELECT id,login,password_hash,created_at FROM users WHERE login=$1`
	if err := s.db.QueryRowContext(ctx, q, login).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return u, nil
}

func (s *PostgresStorage) CreateOrder(ctx context.Context, o *order.Order) error {
	q := `
        INSERT INTO orders (user_id,number,status,uploaded_at)
        VALUES ($1,$2,$3,$4) RETURNING id`
	return s.db.QueryRowContext(ctx, q,
		o.UserID, o.Number, o.Status, o.UploadedAt,
	).Scan(&o.ID)
}

func (s *PostgresStorage) FindOrderByNumber(ctx context.Context, number string) (*order.Order, error) {
	const q = `
    SELECT id, user_id, number, status, accrual_amount, uploaded_at, processed_at
    FROM orders WHERE number = $1`
	var o order.Order
	err := s.db.QueryRowContext(ctx, q, number).
		Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.AccrualAmount, &o.UploadedAt, &o.ProcessedAt)
	if err == sql.ErrNoRows {
		return nil, err
	}
	return &o, err
}

func (s *PostgresStorage) ListOrdersByUser(ctx context.Context, userID int64) ([]order.Order, error) {
	const q = `
        SELECT id, user_id, number, status, accrual_amount, uploaded_at
        FROM orders
        WHERE user_id = $1
        ORDER BY uploaded_at DESC
		`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []order.Order
	for rows.Next() {
		var o order.Order
		var ntTime sql.NullTime
		var ntFloat sql.NullFloat64
		if err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Number,
			&o.Status,
			&ntFloat,
			&o.UploadedAt,
		); err != nil {
			fmt.Println("Something went wrong during scan")
			fmt.Println(err)
			return nil, err
		}
		if ntFloat.Valid {
			o.AccrualAmount = &ntFloat.Float64
		}
		if ntTime.Valid {
			o.ProcessedAt = &ntTime.Time
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) UpdateOrder(ctx context.Context, o *order.Order) error {
	q := `
        UPDATE orders
        SET status=$1, accrual_amount=$2, processed_at=$3
        WHERE id=$4`
	_, err := s.db.ExecContext(ctx, q,
		o.Status, o.AccrualAmount, o.ProcessedAt, o.ID,
	)
	return err
}

func (s *PostgresStorage) ListOrdersForPolling(ctx context.Context) ([]order.Order, error) {
	const q = `
        SELECT number, status, accrual_amount, uploaded_at, processed_at
        FROM orders
        WHERE status IN ('NEW','PROCESSING')
        ORDER BY uploaded_at
    `
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []order.Order
	for rows.Next() {
		var o order.Order
		var accrualNull sql.NullFloat64
		var processedAtNull sql.NullTime

		if err := rows.Scan(&o.Number, &o.Status, &accrualNull, &o.UploadedAt, &processedAtNull); err != nil {
			return nil, err
		}
		if accrualNull.Valid {
			o.AccrualAmount = &accrualNull.Float64
		} else {
			o.AccrualAmount = nil
		}
		if processedAtNull.Valid {
			t := processedAtNull.Time
			o.ProcessedAt = &t
		} else {
			o.ProcessedAt = nil
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) UpdateOrderUpdate(ctx context.Context, number string, status string, accrualAmount *float64, processedAt *time.Time) error {
	const q = `
        UPDATE orders
        SET status = $1,
            accrual_amount = $2,
            processed_at = $3
        WHERE number = $4
    `
	_, err := s.db.ExecContext(ctx, q, status, accrualAmount, processedAt, number)
	return err
}

func (s *PostgresStorage) CreateWithdrawal(ctx context.Context, w *balance.Withdrawal) error {
	q := `
        INSERT INTO withdrawals (user_id, order_number, amount, created_at)
        VALUES ($1,$2,$3, $4) RETURNING id`
	return s.db.QueryRowContext(ctx, q, w.UserID, w.OrderNumber, w.Amount, w.CreatedAt).Scan(&w.ID)
}

func (s *PostgresStorage) ListWithdrawalsByUser(ctx context.Context, userID int64) ([]balance.Withdrawal, error) {
	q := `
        SELECT id,user_id,order_number,amount,created_at
        FROM withdrawals WHERE user_id=$1 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []balance.Withdrawal
	for rows.Next() {
		var w balance.Withdrawal
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Amount, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) GetBalance(ctx context.Context, userID int64) (current float64, withdrawn float64, err error) {
	const qAccrual = `
        SELECT COALESCE(SUM(accrual_amount),0)
        FROM orders
        WHERE user_id=$1 AND status IN ('PROCESSED','PROCESSING')`
	if err := s.db.QueryRowContext(ctx, qAccrual, userID).Scan(&current); err != nil {
		return 0, 0, err
	}
	const qWithdrawn = `
        SELECT COALESCE(SUM(amount),0)
        FROM withdrawals
        WHERE user_id=$1`
	if err := s.db.QueryRowContext(ctx, qWithdrawn, userID).Scan(&withdrawn); err != nil {
		return 0, 0, err
	}
	return current - withdrawn, withdrawn, nil
}
