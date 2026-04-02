package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/d-madiou/fintech-ledger/internal/api"
	"github.com/d-madiou/fintech-ledger/internal/config"
	"github.com/d-madiou/fintech-ledger/internal/ledger"
	"github.com/d-madiou/fintech-ledger/internal/platform/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	fmt.Println("Starting Fintech Ledger System...")

	// 2. Database Connection (Infrastructure)
	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatal("Failed to open driver:", err)
	}
	defer db.Close()

	// Give Postgres a short window to become reachable in containerized startup.
	if err := waitForDB(db, 10, 2*time.Second); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	fmt.Println("Database Connected")

	store := storage.NewPostgresStorage(db)
	if err := store.EnsureSchema(context.Background()); err != nil {
		log.Fatal("Failed to initialize database schema:", err)
	}
	svc := ledger.NewService(store)
	server := api.NewServer(svc)

	http.HandleFunc("/transfer", server.HandleTransfer)
	http.HandleFunc("/wallet", server.HandleGetWallet)

	port := ":" + cfg.AppPort
	fmt.Printf("🌍 Server listening on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func waitForDB(db *sql.DB, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = db.Ping()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}
