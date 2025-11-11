package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/JinFuuMugen/gophermart-ya/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	conn *sql.DB
}

func New(dsn string) (*Database, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}

	return &Database{conn: db}, nil
}

func (db *Database) Migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			login TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			current_balance NUMERIC(12,2) DEFAULT 0 NOT NULL,
			withdrawn NUMERIC(12,2) DEFAULT 0 NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			number TEXT PRIMARY KEY,
			login TEXT NOT NULL REFERENCES users(login),
			status TEXT NOT NULL DEFAULT 'NEW',
			accrual NUMERIC(12,2) DEFAULT 0,
			uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id SERIAL PRIMARY KEY,
			login TEXT NOT NULL REFERENCES users(login),
			order_number TEXT NOT NULL,
			sum NUMERIC(12,2) NOT NULL,
			processed_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, query := range schema {
		if _, err := db.conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("cannot init schema: %w", err)
		}
	}

	return nil
}

//
// Users
//

func (db *Database) RegisterUser(login, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("cannot hash password: %w", err)
	}

	_, err = db.conn.Exec(`INSERT INTO users (login, password_hash) VALUES ($1, $2)`, login, string(hash))
	if err != nil {
		return fmt.Errorf("cannot store user: %w", err)
	}
	return nil
}

func (db *Database) AuthenticateUser(login, password string) (bool, error) {
	row := db.conn.QueryRow(`SELECT password_hash FROM users WHERE login = $1`, login)

	var hash string
	err := row.Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return false, nil
	}

	return true, nil
}

func (db *Database) IsLoginTaken(login string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM users WHERE login = $1`, login).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("cannot check login: %w", err)
	}
	return count > 0, nil
}

//
// Orders
//

func (db *Database) CheckOrderOwner(orderNum, login string) (int, error) {
	var existingLogin string
	err := db.conn.QueryRow(`SELECT login FROM orders WHERE number = $1`, orderNum).Scan(&existingLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return 202, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cannot check order: %w", err)
	}
	if existingLogin == login {
		return 200, nil
	}
	return 409, nil
}

func (db *Database) StoreOrder(orderNum, login string) error {
	_, err := db.conn.Exec(`
		INSERT INTO orders (number, login)
		VALUES ($1, $2)
		ON CONFLICT (number) DO NOTHING
	`, orderNum, login)
	if err != nil {
		return fmt.Errorf("cannot insert order: %w", err)
	}
	return nil
}

func (db *Database) GetOrders(login string) ([]models.Order, error) {
	rows, err := db.conn.Query(`
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE login = $1
		ORDER BY uploaded_at DESC`, login)
	if err != nil {
		return nil, fmt.Errorf("cannot query orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var accrual sql.NullFloat64
		err = rows.Scan(&o.Number, &o.Status, &accrual, &o.UploadedAt)
		if err != nil {
			return nil, err
		}
		if accrual.Valid {
			o.Accrual = &accrual.Float64
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (db *Database) GetPendingOrders() ([]models.Order, error) {
	rows, err := db.conn.Query(`SELECT number FROM orders WHERE status IN ('NEW','PROCESSING')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		rows.Scan(&o.Number)
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (db *Database) UpdateOrderAndBalance(number, status string, accrual float64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("cannot begin tx: %w", err)
	}
	defer tx.Rollback()

	var login string
	if err := tx.QueryRow(`SELECT login FROM orders WHERE number = $1`, number).Scan(&login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("cannot get order login: %w", err)
	}

	_, err = tx.Exec(`UPDATE orders SET status=$1, accrual=$2 WHERE number=$3`, status, accrual, number)
	if err != nil {
		return fmt.Errorf("cannot update order: %w", err)
	}

	if status == string(models.OrderStatusProcessed) && accrual > 0 {
		_, err = tx.Exec(`UPDATE users SET current_balance = current_balance + $1 WHERE login = $2`, accrual, login)
		if err != nil {
			return fmt.Errorf("cannot update balance: %w", err)
		}
	}

	return tx.Commit()
}

//
// Balance
//

func (db *Database) GetBalance(login string) (float64, float64, error) {
	var current, withdrawn float64
	err := db.conn.QueryRow(`SELECT current_balance, withdrawn FROM users WHERE login = $1`, login).Scan(&current, &withdrawn)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get balance: %w", err)
	}
	return current, withdrawn, nil
}

func (db *Database) Withdraw(login, order string, sum float64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("cannot begin tx: %w", err)
	}
	defer tx.Rollback()

	var current float64
	if err := tx.QueryRow(`SELECT current_balance FROM users WHERE login = $1`, login).Scan(&current); err != nil {
		return fmt.Errorf("cannot get balance: %w", err)
	}

	if current < sum {
		return fmt.Errorf("insufficient funds")
	}

	_, err = tx.Exec(`INSERT INTO withdrawals (login, order_number, sum) VALUES ($1, $2, $3)`, login, order, sum)
	if err != nil {
		return fmt.Errorf("cannot insert withdrawal: %w", err)
	}

	_, err = tx.Exec(`UPDATE users SET current_balance = current_balance - $1, withdrawn = withdrawn + $1 WHERE login = $2`, sum, login)
	if err != nil {
		return fmt.Errorf("cannot update balance: %w", err)
	}

	return tx.Commit()
}

func (db *Database) GetWithdrawals(login string) ([]models.Withdrawal, error) {
	rows, err := db.conn.Query(`
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE login = $1
		ORDER BY processed_at DESC`, login)
	if err != nil {
		return nil, fmt.Errorf("cannot query withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		err = rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, rows.Err()
}
