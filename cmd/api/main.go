package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/d-madiou/fintech-ledger/internal/api"
	"github.com/d-madiou/fintech-ledger/internal/ledger"
	"github.com/d-madiou/fintech-ledger/internal/platform/storage"
)

func main() {
	connStr := "postgres://postgres:password@localhost:5432/fintech_ledger?sslmode=disable"

	fmt.Println("Starting Fintech Ledger System...")

	// 2. Database Connection (Infrastructure)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open driver:", err)
	}
	defer db.Close()

	// Verify connection is actually alive
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	fmt.Println("Database Connected")

	store := storage.NewPostgresStorage(db)
	svc := ledger.NewService(store)
	server := api.NewServer(svc)

	http.HandleFunc("/transfer", server.HandleTransfer)
	http.HandleFunc("/wallet", server.HandleGetWallet)

	port := ":8080"
	fmt.Printf("🌍 Server listening on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
