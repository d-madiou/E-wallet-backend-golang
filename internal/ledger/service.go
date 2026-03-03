package ledger

import (
	"context"
	"errors"
	"fmt"
)

type Service struct {
	atomic Atomic
}

type TransferRequest struct {
	FromWalletID WalletID
	ToWalletID   WalletID
	Amount       Money
	ReferenceID  string
}

func NewService(atomic Atomic) *Service {
	return &Service{
		atomic: atomic,
	}
}

func (s *Service) TransferMoney(ctx context.Context, req TransferRequest) error {
	if req.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if req.FromWalletID == req.ToWalletID {
		return errors.New("cannot transfer to self")
	}

	return s.atomic.Run(ctx, func(repo Repository) error {

		sender, err := repo.GetWallet(ctx, req.FromWalletID)
		if err != nil {
			return fmt.Errorf("sender not found: %w", err)
		}

		if sender.Balance < req.Amount {
			return errors.New("insufficient funds")
		}

		receiver, err := repo.GetWallet(ctx, req.ToWalletID)
		if err != nil {
			return fmt.Errorf("receiver not found: %w", err)
		}

		txID := TransactionID(fmt.Sprintf("txn-%d", req.Amount))

		debitEntry := &LedgerEntry{
			ID:            "entry-" + string(txID) + "-dr",
			WalletID:      sender.ID,
			TransactionID: txID,
			Amount:        -req.Amount,
			BalanceAfter:  sender.Balance - req.Amount,
		}
		if err := repo.SaveLedgerEntry(ctx, debitEntry); err != nil {
			return fmt.Errorf("failed to save debit entry: %w", err)
		}

		creditEntry := &LedgerEntry{
			ID:            "entry-" + string(txID) + "-cr",
			WalletID:      receiver.ID,
			TransactionID: txID,
			Amount:        req.Amount,
			BalanceAfter:  receiver.Balance + req.Amount,
		}
		if err := repo.SaveLedgerEntry(ctx, creditEntry); err != nil {
			return fmt.Errorf("failed to save credit entry: %w", err)
		}

		if err := repo.UpdateBalance(ctx, sender.ID, debitEntry.BalanceAfter, sender.Version); err != nil {
			return fmt.Errorf("failed to update sender balance: %w", err)
		}

		if err := repo.UpdateBalance(ctx, receiver.ID, creditEntry.BalanceAfter, receiver.Version); err != nil {
			return fmt.Errorf("failed to update receiver balance: %w", err)
		}

		return nil
	})
}
