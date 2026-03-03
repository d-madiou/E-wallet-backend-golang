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

	// 2. Write the Script (The Expectations)
	// We tell the Cardboard Vault: "If the teller asks for wallet '123', give them this fake data."
	fakeRows := sqlmock.NewRows([]string{"id", "owner_id", "currency", "balance", "updated_at", "version"}).
		AddRow("123", "alice_id", "USD", int64(500), time.Now(), 1)

	mockVault.ExpectBegin()
	mockVault.ExpectQuery("SELECT id, owner_id, currency, balance, updated_at, version FROM wallets WHERE id = \\$1").
		WithArgs("123").
		WillReturnRows(fakeRows)
	mockVault.ExpectCommit()

	// 3. Run the Drill! (The Action)
	// Create the PostgresStorage and use its Run method which handles the transaction
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
	// 1. Build the Cardboard Vault
	fakeDB, mockVault, err := sqlmock.New()
	assert.NoError(t, err)
	defer fakeDB.Close()

	walletID := "wallet_alice"
	newBalance := ledger.Money(4000)
	staleVersion := 1 // We pretend the version in the DB is already 2, so 1 is stale

	// 2. Write the Script (The Expectations)
	mockVault.ExpectBegin()

	// We use regex to match your exact UPDATE SQL query.
	// Note: You might need to adjust the spaces to match your exact query string in postgres.go
	updateQuery := `UPDATE wallets SET balance = \$1, updated_at = NOW\(\), version = version \+ 1 WHERE id = \$2 AND version = \$3`

	// THE MAGIC: We tell the mock to return `sqlmock.NewResult(0, 0)`
	// The first 0 is the insert ID (irrelevant here).
	// The second 0 is the RowsAffected. This simulates the Race Condition!
	mockVault.ExpectExec(updateQuery).
		WithArgs(newBalance, walletID, staleVersion).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Because `UpdateBalance` will return an error, our `storage.Run`
	// should catch it and automatically trigger a ROLLBACK.
	mockVault.ExpectRollback()

	// 3. Run the Drill! (The Action)
	storage := NewPostgresStorage(fakeDB)

	err = storage.Run(context.Background(), func(repo ledger.Repository) error {
		// Attempt the update with the stale version
		return repo.UpdateBalance(context.Background(), ledger.WalletID(walletID), newBalance, staleVersion)
	})

	// 4. Grade the Drill (The Assertions)

	// We expect an error to be returned to the very top!
	assert.Error(t, err)

	// The error message MUST contain our custom concurrency warning
	assert.Contains(t, err.Error(), "concurrent update detected")

	// Verify the mock vault saw exactly the commands we expected (Begin -> Exec -> Rollback)
	assert.NoError(t, mockVault.ExpectationsWereMet())
}
