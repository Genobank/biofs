package bioip

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Genobank/biofs/pkg/biocid"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BioIPAsset represents a BioIP Asset on-chain
type BioIPAsset struct {
	Owner          common.Address
	TokenID        *big.Int
	ConsentState   uint8
	CreatedAt      *big.Int
	RevokedAt      *big.Int
	ContentHash    [32]byte
	DataType       string
	DataSize       *big.Int
	BioCID         [32]byte
	IPAssetID      common.Address
	LicenseTermsID *big.Int
	HasLicense     bool
	ParentTokenID  *big.Int
	ChildTokenIDs  []*big.Int
	Generation     *big.Int
	LicenseTokenID *big.Int
}

// LicenseToken represents a PIL license token
type LicenseToken struct {
	TokenID       *big.Int
	ParentTokenID *big.Int
	MintedFor     common.Address
	MintedAt      *big.Int
	Consumed      bool
	ConsumedBy    *big.Int
}

// BioIPManager handles interactions with BioIPRegistry contract
type BioIPManager struct {
	client          *ethclient.Client
	registryAddress common.Address
	chainRPC        map[string]string
}

// NewBioIPManager creates a new BioIP manager
func NewBioIPManager() *BioIPManager {
	return &BioIPManager{
		chainRPC: map[string]string{
			"story":     "https://rpc.story.foundation",
			"avalanche": "https://api.avax.network/ext/bc/C/rpc",
			"ethereum":  "https://eth.llamarpc.com",
		},
	}
}

// MintRootBioIP creates a new root BioIP with license terms
func (m *BioIPManager) MintRootBioIP(
	ctx context.Context,
	chain string,
	contentHash [32]byte,
	dataType string,
	dataSize uint64,
	bioCID [32]byte,
	ipAssetID common.Address,
	licenseTermsID *big.Int,
	signer *bind.TransactOpts,
) (*big.Int, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call mintRootBioIP
	// For now, return placeholder

	_ = client
	_ = signer

	return big.NewInt(1), nil
}

// MintLicenseTokens mints license tokens for creating derivatives
// MUST be called BEFORE creating the derivative
func (m *BioIPManager) MintLicenseTokens(
	ctx context.Context,
	chain string,
	parentTokenID *big.Int,
	receiver common.Address,
	amount *big.Int,
	signer *bind.TransactOpts,
) ([]*big.Int, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call mintLicenseTokens

	_ = client
	_ = parentTokenID
	_ = receiver
	_ = amount
	_ = signer

	// Return placeholder license token IDs
	return []*big.Int{big.NewInt(1)}, nil
}

// MintDerivativeBioIP creates a child BioIP WITHOUT license terms
func (m *BioIPManager) MintDerivativeBioIP(
	ctx context.Context,
	chain string,
	contentHash [32]byte,
	dataType string,
	dataSize uint64,
	bioCID [32]byte,
	ipAssetID common.Address,
	signer *bind.TransactOpts,
) (*big.Int, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call mintDerivativeBioIP

	_ = client
	_ = signer

	return big.NewInt(2), nil
}

// RegisterDerivative links child as derivative using license token
// This consumes the license token (one-time use)
func (m *BioIPManager) RegisterDerivative(
	ctx context.Context,
	chain string,
	childTokenID *big.Int,
	licenseTokenID *big.Int,
	signer *bind.TransactOpts,
) error {
	client, err := m.getClient(chain)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call registerDerivative

	_ = client
	_ = childTokenID
	_ = licenseTokenID
	_ = signer

	return nil
}

// GetLineage returns all ancestors of a BioIP
func (m *BioIPManager) GetLineage(
	ctx context.Context,
	chain string,
	tokenID *big.Int,
) ([]*big.Int, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call getLineage

	_ = client
	_ = tokenID

	return []*big.Int{}, nil
}

// GetDescendants returns all descendants (children, grandchildren, etc)
func (m *BioIPManager) GetDescendants(
	ctx context.Context,
	chain string,
	tokenID *big.Int,
) ([]*big.Int, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call getDescendants

	_ = client
	_ = tokenID

	return []*big.Int{}, nil
}

// GetAvailableLicenseTokens returns unused license tokens for a parent
func (m *BioIPManager) GetAvailableLicenseTokens(
	ctx context.Context,
	chain string,
	parentTokenID *big.Int,
) ([]*big.Int, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call getAvailableLicenseTokens

	_ = client
	_ = parentTokenID

	return []*big.Int{}, nil
}

// CheckConsent verifies if a wallet has active consent
func (m *BioIPManager) CheckConsent(
	ctx context.Context,
	chain string,
	tokenID *big.Int,
	wallet common.Address,
) (bool, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return false, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call checkConsent

	_ = client
	_ = tokenID
	_ = wallet

	return true, nil
}

// GetBioIP retrieves BioIP asset data
func (m *BioIPManager) GetBioIP(
	ctx context.Context,
	chain string,
	tokenID *big.Int,
) (*BioIPAsset, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call getBioIP

	_ = client
	_ = tokenID

	return &BioIPAsset{
		TokenID:    tokenID,
		Generation: big.NewInt(0),
	}, nil
}

// GetLicenseToken retrieves license token data
func (m *BioIPManager) GetLicenseToken(
	ctx context.Context,
	chain string,
	licenseTokenID *big.Int,
) (*LicenseToken, error) {
	client, err := m.getClient(chain)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", chain, err)
	}

	// TODO: Load contract ABI and call getLicenseToken

	_ = client
	_ = licenseTokenID

	return &LicenseToken{
		TokenID: licenseTokenID,
	}, nil
}

// CreateDerivativeFlow executes the complete derivative creation flow
// This is the recommended way to create derivatives
func (m *BioIPManager) CreateDerivativeFlow(
	ctx context.Context,
	chain string,
	parentTokenID *big.Int,
	childContentHash [32]byte,
	childDataType string,
	childDataSize uint64,
	childBioCID [32]byte,
	childIPAssetID common.Address,
	signer *bind.TransactOpts,
) (*big.Int, error) {
	// Step 1: Mint license token from parent
	licenseTokens, err := m.MintLicenseTokens(
		ctx,
		chain,
		parentTokenID,
		signer.From,
		big.NewInt(1),
		signer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mint license token: %w", err)
	}

	if len(licenseTokens) == 0 {
		return nil, fmt.Errorf("no license tokens minted")
	}

	licenseTokenID := licenseTokens[0]

	// Step 2: Mint child WITHOUT license terms
	childTokenID, err := m.MintDerivativeBioIP(
		ctx,
		chain,
		childContentHash,
		childDataType,
		childDataSize,
		childBioCID,
		childIPAssetID,
		signer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mint derivative: %w", err)
	}

	// Step 3: Register as derivative using license token
	err = m.RegisterDerivative(
		ctx,
		chain,
		childTokenID,
		licenseTokenID,
		signer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register derivative: %w", err)
	}

	return childTokenID, nil
}

// GetLineageTree returns a structured tree of the full lineage
type LineageNode struct {
	TokenID    *big.Int
	BioCID     [32]byte
	DataType   string
	Generation *big.Int
	Children   []*LineageNode
}

func (m *BioIPManager) GetLineageTree(
	ctx context.Context,
	chain string,
	rootTokenID *big.Int,
) (*LineageNode, error) {
	bioip, err := m.GetBioIP(ctx, chain, rootTokenID)
	if err != nil {
		return nil, err
	}

	node := &LineageNode{
		TokenID:    bioip.TokenID,
		BioCID:     bioip.BioCID,
		DataType:   bioip.DataType,
		Generation: bioip.Generation,
		Children:   make([]*LineageNode, 0),
	}

	// Recursively get children
	for _, childID := range bioip.ChildTokenIDs {
		childNode, err := m.GetLineageTree(ctx, chain, childID)
		if err != nil {
			continue // Skip failed children
		}
		node.Children = append(node.Children, childNode)
	}

	return node, nil
}

// getClient returns an ethclient for the specified chain
func (m *BioIPManager) getClient(chain string) (*ethclient.Client, error) {
	rpcURL, ok := m.chainRPC[chain]
	if !ok {
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}

	if m.client != nil {
		return m.client, nil
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	m.client = client
	return client, nil
}

// BioCIDToBioIP converts a BioCID to its corresponding BioIP on-chain
func (m *BioIPManager) BioCIDToBioIP(
	ctx context.Context,
	biocid *biocid.BioCID,
) (*BioIPAsset, error) {
	nftRef := biocid.NFTRef()

	tokenIDBig := new(big.Int)
	tokenIDBig.SetString(nftRef.TokenID, 10)

	return m.GetBioIP(ctx, nftRef.Chain, tokenIDBig)
}