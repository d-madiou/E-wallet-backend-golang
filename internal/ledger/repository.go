package ledger

import (
	"context"
)

type Repository interface {
	CreateTransaction(ctx context.Context, tx *Transaction) error
	GetWallet(ctx context.Context, id WalletID) (*Wallet, error)
	SaveLedgerEntry(ctx context.Context, entry *LedgerEntry) error
	UpdateBalance(ctx context.Context, id WalletID, newBalance Money, expectedVersion int) error
}

type Atomic interface {
	Run(ctx context.Context, fn func(repo Repository) error) error
}
