package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/d-madiou/fintech-ledger/internal/ledger"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) EnsureSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS wallets (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			currency TEXT NOT NULL,
			balance BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			version INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			from_wallet_id TEXT NOT NULL REFERENCES wallets(id),
			to_wallet_id TEXT NOT NULL REFERENCES wallets(id),
			amount BIGINT NOT NULL,
			state TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id TEXT PRIMARY KEY,
			wallet_id TEXT NOT NULL REFERENCES wallets(id),
			transaction_id TEXT NOT NULL,
			amount BIGINT NOT NULL,
			balance_after BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO wallets (id, owner_id, currency, balance, version)
		VALUES
			('wallet_alice', 'alice', 'USD', 10000, 1),
			('wallet_bob', 'bob', 'USD', 10000, 1)
		ON CONFLICT (id) DO NOTHING`,
	}

	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to initialize database schema: %w", err)
		}
	}

	return nil
}

func (s *PostgresStorage) Run(ctx context.Context, fn func(repo ledger.Repository) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	//2. Safety net
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	repo := &postgresRepository{tx: tx}

	if err := fn(repo); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)

	}
	return nil

}

type postgresRepository struct {
	tx *sql.Tx
}

func (r *postgresRepository) CreateTransaction(ctx context.Context, tx *ledger.Transaction) error {
	query := `
		INSERT INTO transactions (id, from_wallet_id, to_wallet_id, amount, state, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`

	operation, err := r.tx.ExecContext(ctx, query, tx.ID, tx.FromWalletID, tx.ToWalletID, tx.Amount, tx.State)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	rowsAffected, err := operation.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("failed to create transaction, no rows affected: %s", tx.ID)
	}

	return nil
}

func (r *postgresRepository) GetWallet(ctx context.Context, id ledger.WalletID) (*ledger.Wallet, error) {
	query := `
	    SELECT id, owner_id, currency, balance, updated_at, version
	    FROM wallets
	    WHERE id = $1
		`

	row := r.tx.QueryRowContext(ctx, query, id)

	var w ledger.Wallet
	err := row.Scan(&w.ID, &w.OwnerID, &w.Currency, &w.Balance, &w.UpdatedAt, &w.Version)
	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("wallet not found: %s", id)
		}
		return nil, fmt.Errorf("failed to scan wallet: %w", err)
	}
	return &w, nil

}

func (r *postgresRepository) UpdateBalance(ctx context.Context, id ledger.WalletID, newBalance ledger.Money, expectedVersion int) error {
	query := `
		UPDATE wallets
		SET balance = $1, updated_at = NOW(), version = version + 1
		WHERE id = $2 AND version = $3
	`
	result, err := r.tx.ExecContext(ctx, query, newBalance, id, expectedVersion)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("concurrent update detected or wallet not found for id: %s", id)
	}
	return nil
}

func (r *postgresRepository) SaveLedgerEntry(ctx context.Context, entry *ledger.LedgerEntry) error {
	query := `
       INSERT INTO ledger_entries (id, wallet_id, transaction_id, amount, balance_after)
       VALUES ($1, $2, $3, $4, $5)
    `

	created, err := r.tx.ExecContext(ctx, query,
		entry.ID,
		entry.WalletID,
		entry.TransactionID,
		entry.Amount,
		entry.BalanceAfter,
	)
	if err != nil {
		return fmt.Errorf("failed to save ledger entry: %w", err)
	}

	rowsAffected, err := created.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("failed to save ledger entry, no rows affected: %s", entry.ID)
	}
	return nil
}

// GetLedgerEntries retrieves ledger entries for a given wallet with a limit
func (r *postgresRepository) GetLedgerEntries(ctx context.Context, walletID ledger.WalletID, limit int) ([]ledger.LedgerEntry, error) {
	query := `
		SELECT id, wallet_id, transaction_id, amount, balance_after 
		FROM ledger_entries 
		WHERE wallet_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`
	rows, err := r.tx.QueryContext(ctx, query, walletID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ledger.LedgerEntry
	for rows.Next() {
		var e ledger.LedgerEntry
		if err := rows.Scan(&e.ID, &e.WalletID, &e.TransactionID, &e.Amount, &e.BalanceAfter); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
