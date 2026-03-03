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

type WalletID string
type TransactionID string

type Wallet struct {
	ID        WalletID
	OwnerID   string
	Currency  Currency
	Balance   Money
	UpdatedAt time.Time
	Version   int
}

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

func (t *Transaction) Validate() error {
	if t.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if t.FromWalletID == t.ToWalletID {
		return errors.New("cannot transfer money to the same wallet")
	}
	return nil
}
