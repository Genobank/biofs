package consent

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Genobank/biofs/pkg/biocid"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ConsentState represents the state of consent for an NFT
type ConsentState int

const (
	ConsentPending ConsentState = iota
	ConsentActive
	ConsentRevoked
	ConsentDeleted
)

// ConsentChecker verifies consent status on-chain
type ConsentChecker struct {
	client   *ethclient.Client
	chainRPC map[string]string // chain name => RPC URL
}

// NewConsentChecker creates a new consent checker
func NewConsentChecker() *ConsentChecker {
	return &ConsentChecker{
		chainRPC: map[string]string{
			"story":     "https://rpc.story.foundation",
			"avalanche": "https://api.avax.network/ext/bc/C/rpc",
			"ethereum":  "https://eth.llamarpc.com",
		},
	}
}

// CheckConsent verifies if a wallet has active consent for an NFT
func (c *ConsentChecker) CheckConsent(ctx context.Context, nftRef biocid.NFTReference, wallet common.Address) (bool, error) {
	// Connect to appropriate chain
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return false, fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	// Get contract instance
	contractAddr := common.HexToAddress(nftRef.Collection)

	// TODO: Load ABI and create contract binding
	// For now, we'll use a simple call

	// Check if wallet owns the NFT or has permission
	hasAccess, err := c.checkOnChainAccess(ctx, client, contractAddr, nftRef.TokenID, wallet)
	if err != nil {
		return false, fmt.Errorf("failed to check on-chain access: %w", err)
	}

	return hasAccess, nil
}

// GetConsentState retrieves the current state of consent for an NFT
func (c *ConsentChecker) GetConsentState(ctx context.Context, nftRef biocid.NFTReference) (ConsentState, error) {
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return ConsentPending, fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	contractAddr := common.HexToAddress(nftRef.Collection)

	// TODO: Call contract to get consent state
	// For now, return placeholder

	_ = client
	_ = contractAddr

	return ConsentActive, nil
}

// WatchConsentEvents listens for consent revocation events
func (c *ConsentChecker) WatchConsentEvents(ctx context.Context, nftRef biocid.NFTReference, callback func(ConsentState)) error {
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	// TODO: Subscribe to contract events
	// Watch for ConsentRevoked, ContentDeleted events

	_ = client
	_ = callback

	return nil
}

// VerifyDeletion verifies that content has been deleted on-chain
func (c *ConsentChecker) VerifyDeletion(ctx context.Context, nftRef biocid.NFTReference) (bool, int, error) {
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return false, 0, fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	contractAddr := common.HexToAddress(nftRef.Collection)

	// TODO: Call contract to check deletion proof
	// Return: (isDeleted, nodeCount, error)

	_ = client
	_ = contractAddr

	return false, 0, nil
}

// getClient returns an ethclient for the specified chain
func (c *ConsentChecker) getClient(chain string) (*ethclient.Client, error) {
	rpcURL, ok := c.chainRPC[chain]
	if !ok {
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}

	if c.client != nil {
		return c.client, nil
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	c.client = client
	return client, nil
}

// checkOnChainAccess checks if wallet has access to NFT
func (c *ConsentChecker) checkOnChainAccess(ctx context.Context, client *ethclient.Client, contract common.Address, tokenID string, wallet common.Address) (bool, error) {
	// Convert tokenID to big.Int
	tokenIDBig := new(big.Int)
	tokenIDBig.SetString(tokenID, 10)

	// TODO: Call checkConsent(tokenId, wallet) on contract
	// For now, return placeholder

	_ = ctx
	_ = contract
	_ = tokenIDBig

	return true, nil
}

// GetOwner returns the owner of an NFT
func (c *ConsentChecker) GetOwner(ctx context.Context, nftRef biocid.NFTReference) (common.Address, error) {
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	contractAddr := common.HexToAddress(nftRef.Collection)

	// Convert tokenID to big.Int
	tokenIDBig := new(big.Int)
	tokenIDBig.SetString(nftRef.TokenID, 10)

	// TODO: Call ownerOf or balanceOf on ERC1155
	// For now, return zero address

	_ = client
	_ = contractAddr
	_ = tokenIDBig

	return common.Address{}, nil
}

// ConsentOptions for creating new consents
type ConsentOptions struct {
	ContentHash []byte
	DataType    string
	DataSize    uint64
	BioCID      string
}

// CreateConsent mints a new NFT and grants consent on-chain
func (c *ConsentChecker) CreateConsent(ctx context.Context, chain string, collection common.Address, opts ConsentOptions, signer *bind.TransactOpts) (string, error) {
	client, err := c.getClient(chain)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Call mintAndGrantConsent on contract
	// Return tokenID as string

	_ = client
	_ = collection
	_ = opts
	_ = signer

	return "1", nil
}

// RevokeConsent revokes consent for an NFT on-chain
func (c *ConsentChecker) RevokeConsent(ctx context.Context, nftRef biocid.NFTReference, signer *bind.TransactOpts) error {
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	contractAddr := common.HexToAddress(nftRef.Collection)

	// TODO: Call revokeConsent(tokenId) on contract

	_ = client
	_ = contractAddr
	_ = signer

	return nil
}

// BurnAndDelete burns NFT and triggers deletion on-chain
func (c *ConsentChecker) BurnAndDelete(ctx context.Context, nftRef biocid.NFTReference, merkleRoot [32]byte, nodeCount *big.Int, signer *bind.TransactOpts) error {
	client, err := c.getClient(nftRef.Chain)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", nftRef.Chain, err)
	}

	contractAddr := common.HexToAddress(nftRef.Collection)

	// TODO: Call burnAndDelete(tokenId, merkleRoot, nodeCount) on contract

	_ = client
	_ = contractAddr
	_ = merkleRoot
	_ = nodeCount
	_ = signer

	return nil
}