package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/desync-labs/tx-manager/executor/internal/domain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var _ TransactionExecutorInterface = (*EVMTransaction)(nil)

type EVMTransaction struct {
	// keyCache cache.KeyCacheInterface
	client          *ethclient.Client
	url             string
	txStatusTimeout int
}

func NewEVMTransaction(url string, txStatusTimeout int) *EVMTransaction {
	evm := &EVMTransaction{
		url:             url,
		txStatusTimeout: txStatusTimeout,
	}
	evm.setupClient()
	return evm

}

func (e *EVMTransaction) setupClient() {
	client, err := ethclient.Dial(e.url)
	if err != nil {
		slog.Error("Failed to connect to the network", "error", err)
	}
	e.client = client
}

func (e *EVMTransaction) Execute(key string, tx *domain.Transaction) (bool, string, error) {
	slog.Info("Executing transaction", "id", tx.Id)
	// === Load Private Key ===
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		slog.Error("Invalid private key", "id", tx.Id)
		return false, "", err
	}

	// === Derive Public Address ===
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		slog.Error("Cannot assert type: publicKey is not of type *ecdsa.PublicKey", "id", tx.Id)
		return false, "", err
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// === Get Nonce ===
	nonce, err := e.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		slog.Error("Failed to get nonce", "id", tx.Id, "error", err)
		return false, "", err
	}

	// === Suggest Gas Price ===
	gasPrice, err := e.client.SuggestGasPrice(context.Background())
	if err != nil {
		slog.Error("Failed to suggest gas price", "id", tx.Id, "error", err)
		return false, "", err
	}

	// === Prepare Transaction ===

	// Convert encodedData to bytes
	encodedData := string(tx.Data)[2:] // Remove the 0x prefix
	dataBytes, err := hex.DecodeString(encodedData)
	if err != nil {
		slog.Error("Invalid encoded data", "id", tx.Id, "error", err)
		return false, "", err
	}

	// Define the contract address
	//TODO: Send the contract address in the transaction
	contractAddr := common.HexToAddress(tx.ContractAddress)

	// Estimate Gas Limit (optional but recommended)
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: dataBytes,
	}

	gasLimit, err := e.client.EstimateGas(context.Background(), msg)
	if err != nil {
		slog.Error("Failed to estimate gas, using default gas limit", "id", tx.Id, "error", err)
		gasLimit = uint64(300000) // Set a default gas limit if estimation fails
	}

	// Get the chain ID
	chainID, err := e.client.NetworkID(context.Background())
	if err != nil {
		slog.Error("Failed to get network for transaction", "id", tx.Id, "error", err)
		return false, "", err

	}

	// Create the transaction
	transaction := types.NewTransaction(nonce, contractAddr, big.NewInt(0), gasLimit, gasPrice, dataBytes)

	// === Sign the Transaction ===
	signedTx, err := types.SignTx(transaction, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		slog.Error("Failed to sign transaction", "id", tx.Id, "error", err)
		return false, signedTx.Hash().String(), err
	}

	// === Send the Transaction ===
	err = e.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		slog.Error("Failed to send transaction", "id", tx.Id, "error", err)
		return false, signedTx.Hash().String(), err
	}

	slog.Info("Transaction sent", "id", tx.Id,
		"from", fromAddress.Hex(),
		"to", contractAddr.Hex(),
		"nonce", nonce,
		"gasPrice", gasPrice,
		"gasLimit", gasLimit,
		"chainID", chainID,
		"txHash", signedTx.Hash().Hex())

	// === Monitor the Transaction ===
	reciept, err := e.monitorTransaction(signedTx.Hash(), time.Duration(e.txStatusTimeout)*time.Second)
	if err != nil {
		slog.Error("Failed to monitor transaction", "id", tx.Id, "error", err)
		return false, signedTx.Hash().String(), err
	}

	return reciept.Status == types.ReceiptStatusSuccessful, signedTx.Hash().String(), nil
}

func (e *EVMTransaction) monitorTransaction(txHash common.Hash, timeoutDuration time.Duration) (*types.Receipt, error) {
	// Create a context with the specified timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// The context has timed out
			slog.Warn("Transaction monitoring timed out", "txHash", txHash.Hex(), "timeout", timeoutDuration)
			return nil, fmt.Errorf("transaction timed out, transaction can still be mined.. txHash: %s", txHash.Hex())

		case <-ticker.C:
			// Attempt to retrieve the transaction receipt
			receipt, err := e.client.TransactionReceipt(ctx, txHash)
			if err != nil {
				if err == ethereum.NotFound {
					// Receipt not yet available
					slog.Debug("Transaction not yet mined", "txHash", txHash.Hex())
				} else {
					// An unexpected error occurred
					slog.Error("Error retrieving receipt", "txHash", txHash.Hex(), "error", err)
					return nil, fmt.Errorf("error retrieving receipt: %v", err)
				}
			} else {
				// Receipt found
				slog.Info("Transaction mined", "txHash", txHash.Hex(), "status", receipt.Status)
				return receipt, nil
			}
		}
	}
}
