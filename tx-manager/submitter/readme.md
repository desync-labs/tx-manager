commands : 
- grpcurl --plaintext localhost:50051 list
- grpcurl --plaintext localhost:50051 describe TransactionSubmitter.SubmitTransaction
- grpcurl --plaintext localhost:50051 describe .TransactionRequest
- grpcurl --plaintext localhost:50051 describe .TransactionResponse
- grpcurl --plaintext \ 
        -d '{"app_name": "oracle-app", "priority": 2, "tx_data": "price-update()"}' \
        localhost:50051 \
        TransactionSubmitter.SubmitTransaction