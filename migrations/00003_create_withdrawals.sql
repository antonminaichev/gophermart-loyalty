-- +goose Up
CREATE TABLE IF NOT EXISTS withdrawals (
            id SERIAL PRIMARY KEY,
            user_id INT NOT NULL REFERENCES users(id),
			order_number TEXT NOT NULL,
            amount DOUBLE PRECISION NOT NULL,
            created_at TIMESTAMPTZ NOT NULL
        );

-- +goose Down
DROP TABLE IF EXISTS withdrawals;