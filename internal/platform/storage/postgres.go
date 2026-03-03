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

	// 4. Run the function with the repository
	if err := fn(repo); err != nil {
		tx.Rollback()
		return err
	}

	// 5. Success, commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)

	}
	return nil

}

// Here we need to define the transition bound repository
type postgresRepository struct {
	tx *sql.Tx
}

func (r *postgresRepository) CreateTransaction(ctx context.Context, tx *ledger.Transaction) error {
	// Let's write the SQL query to create a transaction
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

	// THE FIX: Unpack the struct manually into the 5 placeholders
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
