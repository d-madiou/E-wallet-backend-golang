package ledger

import (
	"errors"
	"time"
)

// 1. Value Objects this is the building block

// note: we never use float64 for money, we use int64 to represent cents
type Money int64

// currency represents the currency code (e.g., USD, EUR)
type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
)

// strong types for IDs to prevent mix-ups
type WalletID string
type TransactionID string

// Let's define the struct for wallet considered as snapshot of the wallet at a point in time
type Wallet struct {
	ID        WalletID
	OwnerID   string   // could be user ID or business ID
	Currency  Currency // currency of the wallet
	Balance   Money    // balance in cents to avoid floating point issues
	UpdatedAt time.Time
	Version   int // for optimistic locking
}

// Transaction represents a transfer of money between wallets
type Transaction struct {
	ID           TransactionID
	FromWalletID WalletID
	ToWalletID   WalletID
	Amount       Money
	Currency     Currency
	State        TransactionState
	CreatedAt    time.Time
}

type TransactionState string

const (
	StatePending   TransactionState = "PENDING"
	StateCompleted TransactionState = "COMPLETED"
	StateFailed    TransactionState = "FAILED"
)

type LedgerEntry struct {
	ID            string
	WalletID      WalletID
	TransactionID TransactionID
	Amount        Money
	BalanceAfter  Money
}

// 2. Domain logic where methods are attached to the entities

// domain logic validation
func (t *Transaction) Validate() error {
	if t.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if t.FromWalletID == t.ToWalletID {
		return errors.New("cannot transfer money to the same wallet")
	}
	return nil
}
