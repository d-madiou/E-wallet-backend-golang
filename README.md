## Real E-wallet-app

go run cmd/api/main.go

curl -X POST http://localhost:8080/transfer \
     -H "Content-Type: application/json" \
     -d '{
           "from_wallet": "wallet_alice",
           "to_wallet": "wallet_bob",
           "amount": -5000
         }'