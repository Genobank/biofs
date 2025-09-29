package biocid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
)

// BioCID represents a Biological Content Identifier
// Format: biocid://v1/<chain>/<collection>/<tokenId>/<contentHash>/<consentSig>
type BioCID struct {
	Version     string // Protocol version (v1)
	Chain       string // EVM chain (story, avalanche, ethereum)
	Collection  string // NFT contract address
	TokenID     string // Token ID
	ContentHash string // SHA256 hash of content
	ConsentSig  string // Owner's consent signature
}

// NFTReference identifies the NFT that gates access to content
type NFTReference struct {
	Chain      string
	Collection string
	TokenID    string
}

// NewBioCID creates a new BioCID from components
func NewBioCID(chain, collection, tokenID string, content []byte, consentSig string) (*BioCID, error) {
	// Validate inputs
	if chain == "" || collection == "" || tokenID == "" {
		return nil, fmt.Errorf("chain, collection, and tokenID are required")
	}

	// Compute content hash
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])

	return &BioCID{
		Version:     "v1",
		Chain:       chain,
		Collection:  collection,
		TokenID:     tokenID,
		ContentHash: contentHash,
		ConsentSig:  consentSig,
	}, nil
}

// ParseBioCID parses a BioCID string
// Format: biocid://v1/<chain>/<collection>/<tokenId>/<contentHash>/<consentSig>
func ParseBioCID(s string) (*BioCID, error) {
	// Remove biocid:// prefix
	if !strings.HasPrefix(s, "biocid://") {
		return nil, fmt.Errorf("invalid biocid: must start with biocid://")
	}

	s = strings.TrimPrefix(s, "biocid://")

	// Split into components
	parts := strings.Split(s, "/")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid biocid format: expected at least 5 parts, got %d", len(parts))
	}

	return &BioCID{
		Version:     parts[0],
		Chain:       parts[1],
		Collection:  parts[2],
		TokenID:     parts[3],
		ContentHash: parts[4],
		ConsentSig:  strings.Join(parts[5:], "/"), // Consent sig may contain /
	}, nil
}

// String returns the BioCID as a string
func (b *BioCID) String() string {
	return fmt.Sprintf("biocid://%s/%s/%s/%s/%s/%s",
		b.Version,
		b.Chain,
		b.Collection,
		b.TokenID,
		b.ContentHash,
		b.ConsentSig,
	)
}

// NFTRef returns the NFT reference from this BioCID
func (b *BioCID) NFTRef() NFTReference {
	return NFTReference{
		Chain:      b.Chain,
		Collection: b.Collection,
		TokenID:    b.TokenID,
	}
}

// ToMultihash converts BioCID to a multihash (for DHT)
func (b *BioCID) ToMultihash() (multihash.Multihash, error) {
	// Create unique identifier from BioCID components
	identifier := fmt.Sprintf("%s:%s:%s:%s",
		b.Chain,
		b.Collection,
		b.TokenID,
		b.ContentHash,
	)

	// Hash the identifier
	hash := sha256.Sum256([]byte(identifier))

	// Create multihash
	mh, err := multihash.Encode(hash[:], multihash.SHA2_256)
	if err != nil {
		return nil, fmt.Errorf("failed to create multihash: %w", err)
	}

	return mh, nil
}

// ToBase58 returns the BioCID encoded as base58
func (b *BioCID) ToBase58() (string, error) {
	mh, err := b.ToMultihash()
	if err != nil {
		return "", err
	}

	encoded, err := multibase.Encode(multibase.Base58BTC, mh)
	if err != nil {
		return "", fmt.Errorf("failed to encode as base58: %w", err)
	}

	return encoded, nil
}

// Validate checks if the BioCID is valid
func (b *BioCID) Validate() error {
	if b.Version != "v1" {
		return fmt.Errorf("unsupported version: %s", b.Version)
	}

	if b.Chain == "" {
		return fmt.Errorf("chain is required")
	}

	validChains := map[string]bool{
		"story":     true,
		"avalanche": true,
		"ethereum":  true,
	}
	if !validChains[b.Chain] {
		return fmt.Errorf("unsupported chain: %s", b.Chain)
	}

	if !strings.HasPrefix(b.Collection, "0x") || len(b.Collection) != 42 {
		return fmt.Errorf("invalid collection address: %s", b.Collection)
	}

	if b.TokenID == "" {
		return fmt.Errorf("tokenID is required")
	}

	if len(b.ContentHash) != 64 { // SHA256 hex = 64 chars
		return fmt.Errorf("invalid content hash length: expected 64, got %d", len(b.ContentHash))
	}

	if !strings.HasPrefix(b.ConsentSig, "0x") {
		return fmt.Errorf("invalid consent signature: must start with 0x")
	}

	return nil
}

// Equal checks if two BioCIDs are equal
func (b *BioCID) Equal(other *BioCID) bool {
	return b.Version == other.Version &&
		b.Chain == other.Chain &&
		b.Collection == other.Collection &&
		b.TokenID == other.TokenID &&
		b.ContentHash == other.ContentHash &&
		b.ConsentSig == other.ConsentSig
}

// VerifyContent verifies that content matches the hash in BioCID
func (b *BioCID) VerifyContent(content []byte) bool {
	hash := sha256.Sum256(content)
	computedHash := hex.EncodeToString(hash[:])
	return computedHash == b.ContentHash
}

// NFTReference methods

// String returns the NFT reference as a string
func (n NFTReference) String() string {
	return fmt.Sprintf("%s/%s/%s", n.Chain, n.Collection, n.TokenID)
}

// ParseNFTRef parses an NFT reference string
func ParseNFTRef(s string) (NFTReference, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return NFTReference{}, fmt.Errorf("invalid nft reference format")
	}

	return NFTReference{
		Chain:      parts[0],
		Collection: parts[1],
		TokenID:    parts[2],
	}, nil
}

// ToBiofsURI converts BioCID to a biofs:// URI
// Format: biofs://<chain>/<collection>/<tokenId>/<path>
func (b *BioCID) ToBiofsURI(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return fmt.Sprintf("biofs://%s/%s/%s%s",
		b.Chain,
		b.Collection,
		b.TokenID,
		path,
	)
}

// ParseBiofsURI parses a biofs:// URI and returns NFT reference and path
func ParseBiofsURI(uri string) (NFTReference, string, error) {
	// Remove biofs:// prefix
	if !strings.HasPrefix(uri, "biofs://") {
		return NFTReference{}, "", fmt.Errorf("invalid biofs URI: must start with biofs://")
	}

	uri = strings.TrimPrefix(uri, "biofs://")

	// Split into parts
	parts := strings.SplitN(uri, "/", 4)
	if len(parts) < 3 {
		return NFTReference{}, "", fmt.Errorf("invalid biofs URI format")
	}

	nftRef := NFTReference{
		Chain:      parts[0],
		Collection: parts[1],
		TokenID:    parts[2],
	}

	path := "/"
	if len(parts) == 4 {
		path = "/" + parts[3]
	}

	return nftRef, path, nil
}

// DerivativeInfo represents derivative relationship metadata
type DerivativeInfo struct {
	ParentBioCID   *BioCID   // Parent BioCID (nil if root)
	ChildBioCIDs   []*BioCID // Child BioCIDs
	Generation     int       // 0=root, 1=child, 2=grandchild, etc
	LicenseTokenID string    // License token used to create this derivative
	LicenseTermsID string    // PIL license terms ID
}

// IsRoot returns true if this is a root BioIP (no parent)
func (d *DerivativeInfo) IsRoot() bool {
	return d.ParentBioCID == nil && d.Generation == 0
}

// HasChildren returns true if this BioIP has derivatives
func (d *DerivativeInfo) HasChildren() bool {
	return len(d.ChildBioCIDs) > 0
}

// ChildCount returns the number of direct children
func (d *DerivativeInfo) ChildCount() int {
	return len(d.ChildBioCIDs)
}

// NewDerivativeInfo creates a new DerivativeInfo
func NewDerivativeInfo(generation int) *DerivativeInfo {
	return &DerivativeInfo{
		Generation:   generation,
		ChildBioCIDs: make([]*BioCID, 0),
	}
}

// AddChild adds a child BioCID to the derivative info
func (d *DerivativeInfo) AddChild(child *BioCID) {
	d.ChildBioCIDs = append(d.ChildBioCIDs, child)
}

// SetParent sets the parent BioCID
func (d *DerivativeInfo) SetParent(parent *BioCID) {
	d.ParentBioCID = parent
}

// LineageMetadata represents complete lineage information
type LineageMetadata struct {
	Self       *BioCID   // Current BioCID
	Ancestors  []*BioCID // All ancestors (parent, grandparent, etc)
	Descendants []*BioCID // All descendants (children, grandchildren, etc)
	Generation int       // Generation number (0=root)
}

// GetAncestorCount returns number of ancestors
func (l *LineageMetadata) GetAncestorCount() int {
	return len(l.Ancestors)
}

// GetDescendantCount returns number of descendants
func (l *LineageMetadata) GetDescendantCount() int {
	return len(l.Descendants)
}

// IsRoot returns true if this is a root with no ancestors
func (l *LineageMetadata) IsRoot() bool {
	return len(l.Ancestors) == 0 && l.Generation == 0
}

// GetRoot returns the root ancestor (generation 0)
func (l *LineageMetadata) GetRoot() *BioCID {
	if l.IsRoot() {
		return l.Self
	}
	if len(l.Ancestors) > 0 {
		return l.Ancestors[len(l.Ancestors)-1]
	}
	return nil
}

// GetParent returns the immediate parent (if any)
func (l *LineageMetadata) GetParent() *BioCID {
	if len(l.Ancestors) > 0 {
		return l.Ancestors[0]
	}
	return nil
}