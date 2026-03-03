package storage

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/d-madiou/fintech-ledger/internal/ledger"
	"github.com/stretchr/testify/assert"
)

func TestGetWallet_Success(t *testing.T) {
	// 1. Build the Cardboard Vault (The Setup)
	fakeDB, mockVault, _ := sqlmock.New()
	defer fakeDB.Close()

	fakeRows := sqlmock.NewRows([]string{"id", "owner_id", "currency", "balance", "updated_at", "version"}).
		AddRow("123", "alice_id", "USD", int64(500), time.Now(), 1)

	mockVault.ExpectBegin()
	mockVault.ExpectQuery("SELECT id, owner_id, currency, balance, updated_at, version FROM wallets WHERE id = \\$1").
		WithArgs("123").
		WillReturnRows(fakeRows)
	mockVault.ExpectCommit()

	storage := NewPostgresStorage(fakeDB)

	var retrievedWallet *ledger.Wallet
	var retrieveErr error

	err := storage.Run(context.Background(), func(repo ledger.Repository) error {
		var e error
		retrievedWallet, e = repo.GetWallet(context.Background(), "123")
		retrieveErr = e
		return e
	})

	// 4. Grade the Drill with our Clipboard (The Assertions)
	assert.NoError(t, err)
	assert.NoError(t, retrieveErr)
	assert.NotNil(t, retrievedWallet)
	assert.Equal(t, ledger.Money(500), retrievedWallet.Balance)
}

func TestUpdateBalance_ConcurrentUpdateDetected(t *testing.T) {
	fakeDB, mockVault, err := sqlmock.New()
	assert.NoError(t, err)
	defer fakeDB.Close()

	walletID := "wallet_alice"
	newBalance := ledger.Money(4000)
	staleVersion := 1
	mockVault.ExpectBegin()

	updateQuery := `UPDATE wallets SET balance = \$1, updated_at = NOW\(\), version = version \+ 1 WHERE id = \$2 AND version = \$3`

	mockVault.ExpectExec(updateQuery).
		WithArgs(newBalance, walletID, staleVersion).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mockVault.ExpectRollback()

	storage := NewPostgresStorage(fakeDB)

	err = storage.Run(context.Background(), func(repo ledger.Repository) error {

		return repo.UpdateBalance(context.Background(), ledger.WalletID(walletID), newBalance, staleVersion)
	})

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "concurrent update detected")

	assert.NoError(t, mockVault.ExpectationsWereMet())
}
