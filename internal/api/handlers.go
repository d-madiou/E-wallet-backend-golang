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

func (s *Server) HandleGetWallet(w http.ResponseWriter, r *http.Request) {
	// 1. Ensure it's a GET request
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Extract wallet ID from query parameters (Your excellent idea!)
	walletID := r.URL.Query().Get("id")
	if walletID == "" {
		http.Error(w, "Missing wallet ID", http.StatusBadRequest)
		return
	}

	// 3. Catch ALL THREE return values from the service layer
	wallet, entries, err := s.LedgerService.GetWalletStatement(r.Context(), ledger.WalletID(walletID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// 4. Create a DTO (Data Transfer Object) to format the JSON perfectly
	// We map our internal domain structs to the exact JSON the frontend needs
	type transactionDTO struct {
		ID           string `json:"id"`
		Amount       int64  `json:"amount"`
		BalanceAfter int64  `json:"balance_after"`
	}

	type statementResponse struct {
		WalletID     string           `json:"wallet_id"`
		Balance      int64            `json:"balance"`
		Transactions []transactionDTO `json:"transactions"`
	}

	// 5. Populate the response
	resp := statementResponse{
		WalletID:     string(wallet.ID),
		Balance:      int64(wallet.Balance),
		Transactions: make([]transactionDTO, 0),
	}

	// Convert domain ledger entries to our API DTO
	for _, entry := range entries {
		resp.Transactions = append(resp.Transactions, transactionDTO{
			ID:           entry.ID,
			Amount:       int64(entry.Amount),
			BalanceAfter: int64(entry.BalanceAfter),
		})
	}

	// 6. Send it back!
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
