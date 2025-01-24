package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"log/slog"
	"math/big"

	"github.com/desync-labs/tx-manager/executor/internal/domain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EVMTransaction struct {
	client *ethclient.Client
	url    string
}

func NewEVMTransaction(url string) *EVMTransaction {
	evm := &EVMTransaction{
		url: url,
	}
	evm.setupClient()
	return evm

}

func (e *EVMTransaction) setupClient() {
	client, err := ethclient.Dial(e.url)
	if err != nil {
		slog.Error("Failed to connect to the network: %v", err, "network", "apothem")
	}
	e.client = client
}

func (e *EVMTransaction) Execute(key string, tx *domain.Transaction) error {
	slog.Info("Executing transaction", "id", tx.Id)
	// === Load Private Key ===
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		slog.Error("Invalid private key", "id", tx.Id)
		return err
	}

	// === Derive Public Address ===
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		slog.Error("Cannot assert type: publicKey is not of type *ecdsa.PublicKey", "id", tx.Id)
		return err
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// === Get Nonce ===
	nonce, err := e.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		slog.Error("Failed to get nonce", "id", tx.Id, "error", err)
		return err
	}

	// === Suggest Gas Price ===
	gasPrice, err := e.client.SuggestGasPrice(context.Background())
	if err != nil {
		slog.Error("Failed to suggest gas price", "id", tx.Id, "error", err)
		return err
	}

	// === Prepare Transaction ===

	// Convert encodedData to bytes
	encodedData := string(tx.Data)[2:] // Remove the 0x prefix
	dataBytes, err := hex.DecodeString(encodedData)
	if err != nil {
		slog.Error("Invalid encoded data", "id", tx.Id, "error", err)
		return err
	}

	// Define the contract address
	//TODO: Send the contract address in the transaction
	contractAddr := common.HexToAddress("0x9BDcECf207d1007a03fB3aE7EE4030b4BD3b9803")

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
		return err

	}

	// Create the transaction
	transaction := types.NewTransaction(nonce, contractAddr, big.NewInt(0), gasLimit, gasPrice, dataBytes)

	// === Sign the Transaction ===
	signedTx, err := types.SignTx(transaction, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		slog.Error("Failed to sign transaction", "id", tx.Id, "error", err)
		return err
	}

	// === Send the Transaction ===
	err = e.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		slog.Error("Failed to send transaction", "id", tx.Id, "error", err)
		return err
	}

	slog.Info("Transaction sent", "id", tx.Id,
		"from", fromAddress.Hex(),
		"to", contractAddr.Hex(),
		"nonce", nonce,
		"gasPrice", gasPrice,
		"gasLimit", gasLimit,
		"chainID", chainID,
		"txHash", signedTx.Hash().Hex())

	return nil
}
