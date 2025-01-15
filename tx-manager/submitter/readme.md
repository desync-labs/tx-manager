commands : 
- grpcurl --plaintext localhost:50051 list
- grpcurl --plaintext localhost:50051 describe TransactionSubmitter.SubmitTransaction
- grpcurl --plaintext localhost:50051 describe .TransactionRequest
- grpcurl --plaintext localhost:50051 describe .TransactionResponse
- grpcurl --plaintext \ 
        -d '{"app_name": "oracle-app", "priority": 2, "tx_data": "price-update()"}' \
        localhost:50051 \
        TransactionSubmitter.SubmitTransaction

- grpcurl --plaintext -d '{"app_name": "price—update-app", "priority": 1, "tx_data": "0xbcf8ecf300000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002584443000000000000000000000000000000000000000000000000000000000043474f0000000000000000000000000000000000000000000000000000000000"}' localhost:50051 TransactionSubmitter.SubmitTransaction       