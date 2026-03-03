package api

import (
	"encoding/json"
	"net/http"

	"github.com/d-madiou/fintech-ledger/internal/ledger"
)

// Let's define the API handlers like a server struct
type Server struct {
	LedgerService *ledger.Service
}

// let's define the newserver fynction to create a new server instance
func NewServer(svc *ledger.Service) *Server {
	return &Server{
		LedgerService: svc,
	}
}

// TransferRequestDTO (Data Transfer Object) defines the exact JSON we expect from the client.
type TransferRequestDTO struct {
	FromWallet string `json:"from_wallet"`
	ToWallet   string `json:"to_wallet"`
	Amount     int64  `json:"amount"`
}

// TransferHandler is the HTTP handler for processing transfer requests for the API POST /transfer
func (s *Server) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	// 1. lET'S ensure it's a POST request for security and correctness
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Parse the JSON body into our DTO
	var reqDTO TransferRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest) //400
		return
	}

	// 3. Map the HTTP DTO to our internal Domain Request
	domainReq := ledger.TransferRequest{
		FromWalletID: ledger.WalletID(reqDTO.FromWallet),
		ToWalletID:   ledger.WalletID(reqDTO.ToWallet),
		Amount:       ledger.Money(reqDTO.Amount),
		ReferenceID:  "http-req-" + r.Header.Get("X-Request-ID"),
	}

	// 4. Let's call the engine
	err := s.LedgerService.TransferMoney(r.Context(), domainReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Transfer completed successfully",
	})
}
