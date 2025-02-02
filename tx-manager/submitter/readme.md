commands : 
- grpcurl --plaintext localhost:50051 list
- grpcurl --plaintext localhost:50051 describe TransactionSubmitter.SubmitTransaction
- grpcurl --plaintext localhost:50051 describe .TransactionRequest
- grpcurl --plaintext localhost:50051 describe .TransactionResponse
- grpcurl --plaintext \ 
        -d '{"app_name": "oracle-app", "priority": 2, "tx_data": "price-update()" "network": 51, "contract_address": "0x9BDcECf207d1007a03fB3aE7EE4030b4BD3b9803"}' \
        localhost:50051 \
        TransactionSubmitter.SubmitTransaction

- grpcurl --plaintext -d '{"app_name": "price—update-app", "priority": 1, "network": 51,"contract_address": "0x9BDcECf207d1007a03fB3aE7EE4030b4BD3b9803","tx_data": "0xbcf8ecf300000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002584443000000000000000000000000000000000000000000000000000000000043474f0000000000000000000000000000000000000000000000000000000000"}' localhost:50051 TransactionSubmitter.SubmitTransaction

grpcurl --plaintext -d '{"app_name": "oracle-app", "priority": 2, "tx_data": "0xbcf8ecf300000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002584443000000000000000000000000000000000000000000000000000000000043474f0000000000000000000000000000000000000000000000000000000000", "network": 51, "contract_address": "0x9BDcECf207d1007a03fB3aE7EE4030b4BD3b9803"}' -v -proto ../submitter/protos/transaction.proto localhost:50051 TransactionSubmitter.SubmitTransactionStream