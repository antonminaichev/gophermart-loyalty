-- +goose Up
CREATE TABLE IF NOT EXISTS orders (
            id SERIAL PRIMARY KEY,
            user_id INT NOT NULL REFERENCES users(id),
            number TEXT UNIQUE NOT NULL,
            status TEXT NOT NULL,
            accrual_amount DOUBLE PRECISION,
            uploaded_at TIMESTAMPTZ NOT NULL,
            processed_at TIMESTAMPTZ
        );

-- +goose Down
DROP TABLE IF EXISTS orders;