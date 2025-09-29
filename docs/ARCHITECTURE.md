# BioFS Architecture
## Biological Filesystem Protocol - GDPR-Compliant Distributed Storage

**Version**: 1.0.0
**Date**: September 29, 2025
**Status**: Design Phase

---

## Executive Summary

BioFS (Biological Filesystem) is a distributed, NFT-gated filesystem protocol inspired by IPFS but designed specifically for genomic data with built-in GDPR/CCPA compliance through consent-driven deletion mechanisms.

### Key Differences from IPFS

| Feature | IPFS | BioFS |
|---------|------|-------|
| **Immutability** | Permanent | Consent-revocable |
| **Access Control** | Public | NFT-gated (ERC1155) |
| **Deletion** | Not possible | Cryptographically verified |
| **Mounting** | read-only | read/write with permissions |
| **Content ID** | CID (hash only) | BioCID (hash + NFT + consent) |
| **Use Case** | General data | Genomic/medical data |

---

## Core Architecture

### 1. BioCID (Biological Content Identifier)

```
biocid://<version>/<chain>/<nft-contract>/<token-id>/<content-hash>/<consent-sig>
```

**Example**:
```
biocid://v1/story/0xC919.../1234/QmXoypiz.../0xa5141ae...
```

**Components**:
- `version`: Protocol version (v1)
- `chain`: EVM chain (story, avalanche, ethereum)
- `nft-contract`: ERC1155 contract address
- `token-id`: NFT token ID
- `content-hash`: SHA256 of content
- `consent-sig`: Owner's consent signature

### 2. Mount Points

```bash
# Mount user's NFT-gated genomic files
sudo biofs mount biofs://story/0xC919.../1234 /mnt/biofs/my-genome

# List files
ls /mnt/biofs/my-genome/
  genome.vcf
  annotation.sqlite
  report.pdf

# Access with automatic NFT verification
cat /mnt/biofs/my-genome/genome.vcf
# ✅ NFT ownership verified
# ✅ Consent active
# → Content delivered
```

### 3. System Components

```
┌─────────────────────────────────────────────────┐
│              Linux Kernel                       │
├─────────────────────────────────────────────────┤
│              FUSE Layer                         │
├─────────────────────────────────────────────────┤
│           biofs-fuse Driver                     │
│  ┌────────────────────────────────────────┐    │
│  │  File Operations:                      │    │
│  │  - open()  → NFT ownership check       │    │
│  │  - read()  → Content retrieval         │    │
│  │  - write() → Consent verification      │    │
│  │  - unlink() → Consent revocation       │    │
│  └────────────────────────────────────────┘    │
├─────────────────────────────────────────────────┤
│           biofs-daemon                          │
│  ┌────────────────────────────────────────┐    │
│  │  Core Services:                        │    │
│  │  - Content Storage (pinning)           │    │
│  │  - P2P Networking (DHT)                │    │
│  │  - NFT Verification                    │    │
│  │  - Consent Monitoring                  │    │
│  │  - Cryptographic Deletion              │    │
│  └────────────────────────────────────────┘    │
├─────────────────────────────────────────────────┤
│          Consent Smart Contracts                │
│  ┌────────────────────────────────────────┐    │
│  │  ConsentRegistry.sol (ERC1155)         │    │
│  │  - grantConsent()                      │    │
│  │  - revokeConsent()                     │    │
│  │  - burnAndDelete()                     │    │
│  │  - checkConsent(nft, wallet)           │    │
│  └────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

---

## Protocol Specification

### Content Storage Flow

1. **Upload**:
```bash
biofs add genome.vcf --nft story/0xC919.../1234 --sign 0xa5141...
```

2. **Daemon Processing**:
   - Compute content hash: `hash(genome.vcf)`
   - Verify NFT ownership on-chain
   - Get consent signature from wallet
   - Create BioCID: `biocid://v1/story/0xC919.../1234/QmHash.../0xSig...`
   - Pin content locally
   - Announce to DHT with consent metadata
   - Store in local blockstore

3. **Network Propagation**:
   - DHT peers discover content
   - Authorized nodes (with verified permissions) can pin
   - Consent state propagated via gossip protocol

### Content Access Flow

1. **Mount Request**:
```bash
sudo biofs mount biofs://story/0xC919.../1234 /mnt/genome
```

2. **FUSE Driver**:
   - Receives mount request
   - Connects to biofs-daemon
   - Verifies user's Web3 signature
   - Checks NFT ownership on-chain
   - Creates virtual filesystem at `/mnt/genome`

3. **File Access**:
```bash
cat /mnt/genome/genome.vcf
```

4. **Access Verification**:
   - FUSE intercepts `open()` syscall
   - Daemon checks consent on-chain
   - Daemon verifies NFT ownership
   - If authorized: retrieve content from local store or DHT peers
   - Return file descriptor to kernel

### Consent Revocation Flow

1. **User Revokes**:
```bash
biofs revoke-consent --nft story/0xC919.../1234 --sign 0xa5141...
```

2. **Smart Contract**:
```solidity
function revokeConsent(uint256 tokenId) external onlyOwner {
    require(ownerOf(tokenId) == msg.sender, "Not owner");
    consentStates[tokenId] = ConsentState.REVOKED;
    emit ConsentRevoked(tokenId, msg.sender, block.timestamp);
}
```

3. **Network Propagation**:
   - Event emitted on-chain
   - All biofs-daemons listening to events
   - Gossip protocol spreads revocation to all nodes
   - Each node verifies signature and unpins content

4. **Cryptographic Deletion**:
   - Content overwritten with random data
   - Blocks removed from local store
   - DHT entries purged
   - Merkle proofs updated
   - Deletion receipt generated: `hash(deleted_content + timestamp + node_sig)`

5. **GDPR Compliance**:
   - Deletion verifiable on-chain
   - Nodes provide deletion receipts
   - Merkle tree proves absence of content

---

## Smart Contracts

### ConsentRegistry.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";

contract ConsentRegistry is ERC1155 {
    enum ConsentState { ACTIVE, REVOKED, DELETED }

    struct ConsentMetadata {
        address owner;
        uint256 tokenId;
        ConsentState state;
        uint256 createdAt;
        uint256 revokedAt;
        bytes32 contentHash;
    }

    mapping(uint256 => ConsentMetadata) public consents;
    mapping(address => mapping(uint256 => bool)) public permittedWallets;

    event ConsentGranted(uint256 indexed tokenId, address indexed owner, bytes32 contentHash);
    event ConsentRevoked(uint256 indexed tokenId, address indexed owner, uint256 timestamp);
    event ContentDeleted(uint256 indexed tokenId, bytes32 deletionProof);
    event PermissionGranted(uint256 indexed tokenId, address indexed wallet);

    function grantConsent(uint256 tokenId, bytes32 contentHash) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");

        consents[tokenId] = ConsentMetadata({
            owner: msg.sender,
            tokenId: tokenId,
            state: ConsentState.ACTIVE,
            createdAt: block.timestamp,
            revokedAt: 0,
            contentHash: contentHash
        });

        emit ConsentGranted(tokenId, msg.sender, contentHash);
    }

    function revokeConsent(uint256 tokenId) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");
        require(consents[tokenId].state == ConsentState.ACTIVE, "Already revoked");

        consents[tokenId].state = ConsentState.REVOKED;
        consents[tokenId].revokedAt = block.timestamp;

        emit ConsentRevoked(tokenId, msg.sender, block.timestamp);
    }

    function burnAndDelete(uint256 tokenId, bytes32 deletionProof) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");

        // Burn NFT
        _burn(msg.sender, tokenId, 1);

        // Mark as deleted
        consents[tokenId].state = ConsentState.DELETED;

        emit ContentDeleted(tokenId, deletionProof);
    }

    function checkConsent(uint256 tokenId, address wallet) external view returns (bool) {
        ConsentMetadata memory consent = consents[tokenId];

        if (consent.state != ConsentState.ACTIVE) return false;
        if (consent.owner == wallet) return true;
        if (permittedWallets[wallet][tokenId]) return true;

        return false;
    }

    function grantPermission(uint256 tokenId, address wallet) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");
        permittedWallets[wallet][tokenId] = true;
        emit PermissionGranted(tokenId, wallet);
    }
}
```

---

## FUSE Implementation

### biofs-fuse Driver (Go)

```go
package biofsfuse

import (
    "context"
    "syscall"

    "github.com/hanwen/go-fuse/v2/fs"
    "github.com/hanwen/go-fuse/v2/fuse"
)

type BioFSRoot struct {
    fs.Inode
    daemon *BioFSDaemon
    nftRef NFTReference
}

type NFTReference struct {
    Chain      string
    Collection string
    TokenID    string
}

// Implement FUSE operations
func (r *BioFSRoot) OnAdd(ctx context.Context) {
    // Initialize connection to biofs-daemon
    r.daemon.Connect()
}

func (r *BioFSRoot) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
    // Check NFT ownership
    if !r.daemon.VerifyNFTOwnership(r.nftRef) {
        return nil, syscall.EACCES
    }

    // Check consent
    if !r.daemon.CheckConsent(r.nftRef) {
        return nil, syscall.ENOENT // File "doesn't exist" if consent revoked
    }

    // Retrieve file metadata from DHT
    fileInfo, err := r.daemon.GetFileInfo(r.nftRef, name)
    if err != nil {
        return nil, syscall.ENOENT
    }

    // Create file node
    node := &BioFSFile{
        daemon: r.daemon,
        nftRef: r.nftRef,
        name:   name,
        size:   fileInfo.Size,
    }

    return r.NewInode(ctx, node, fs.StableAttr{Mode: fuse.S_IFREG}), 0
}

type BioFSFile struct {
    fs.Inode
    daemon *BioFSDaemon
    nftRef NFTReference
    name   string
    size   uint64
}

func (f *BioFSFile) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
    // Verify access every time file is opened
    if !f.daemon.VerifyAccess(f.nftRef, getCurrentWallet()) {
        return nil, 0, syscall.EACCES
    }

    // Return file handle
    return &BioFSFileHandle{
        file:   f,
        daemon: f.daemon,
    }, fuse.FOPEN_KEEP_CACHE, 0
}

type BioFSFileHandle struct {
    file   *BioFSFile
    daemon *BioFSDaemon
    offset int64
}

func (fh *BioFSFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
    // Verify consent is still active
    if !fh.daemon.CheckConsent(fh.file.nftRef) {
        return nil, syscall.ENOENT
    }

    // Retrieve content chunks from DHT
    data, err := fh.daemon.ReadContent(fh.file.nftRef, fh.file.name, off, len(dest))
    if err != nil {
        return nil, syscall.EIO
    }

    return fuse.ReadResultData(data), 0
}
```

---

## P2P Networking

### DHT (Distributed Hash Table)

BioFS uses libp2p DHT with consent-aware routing:

```go
type ConsentAwareDHT struct {
    dht *kad.DHT
    consentChecker *ConsentChecker
}

func (d *ConsentAwareDHT) Provide(ctx context.Context, biocid BioCID) error {
    // Verify we have consent to provide this content
    if !d.consentChecker.VerifyConsent(biocid.NFTRef) {
        return ErrConsentRevoked
    }

    // Announce to DHT
    return d.dht.Provide(ctx, biocid.ContentHash, true)
}

func (d *ConsentAwareDHT) FindProviders(ctx context.Context, biocid BioCID) ([]peer.AddrInfo, error) {
    // Find providers
    providers, err := d.dht.FindProviders(ctx, biocid.ContentHash)
    if err != nil {
        return nil, err
    }

    // Filter to only authorized providers
    authorized := []peer.AddrInfo{}
    for _, provider := range providers {
        if d.isAuthorizedProvider(provider, biocid.NFTRef) {
            authorized = append(authorized, provider)
        }
    }

    return authorized, nil
}
```

### Gossip Protocol for Consent State

```go
type ConsentGossip struct {
    pubsub *pubsub.PubSub
    topic  *pubsub.Topic
}

func (g *ConsentGossip) PublishRevocation(nftRef NFTReference) error {
    msg := &ConsentRevocationMessage{
        Chain:      nftRef.Chain,
        Collection: nftRef.Collection,
        TokenID:    nftRef.TokenID,
        Timestamp:  time.Now().Unix(),
        Signature:  signMessage(nftRef),
    }

    data, _ := json.Marshal(msg)
    return g.topic.Publish(context.Background(), data)
}

func (g *ConsentGossip) Subscribe() {
    sub, _ := g.topic.Subscribe()

    go func() {
        for {
            msg, err := sub.Next(context.Background())
            if err != nil {
                continue
            }

            // Handle revocation
            var revocation ConsentRevocationMessage
            json.Unmarshal(msg.Data, &revocation)

            // Verify signature
            if verifySignature(revocation) {
                // Unpin content immediately
                unpinContent(revocation.NFTRef)
            }
        }
    }()
}
```

---

## CLI Commands

```bash
# Initialize BioFS daemon
biofs init

# Start daemon
biofs daemon

# Add content with NFT reference
biofs add genome.vcf \
  --nft story/0xC91940118822D247B46d1eBA6B7Ed2A16F3aDC36/1234 \
  --wallet 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb4 \
  --sign 0xa5141ae...

# Mount NFT content
sudo biofs mount biofs://story/0xC919.../1234 /mnt/my-genome

# List files
biofs ls biofs://story/0xC919.../1234

# Get file
biofs cat biofs://story/0xC919.../1234/genome.vcf

# Grant permission to another wallet
biofs grant-permission \
  --nft story/0xC919.../1234 \
  --to 0x123... \
  --sign 0xa5141...

# Revoke consent (triggers global unpin)
biofs revoke-consent \
  --nft story/0xC919.../1234 \
  --sign 0xa5141...

# Verify deletion
biofs verify-deletion --nft story/0xC919.../1234

# Unmount
sudo biofs unmount /mnt/my-genome
```

---

## Integration with NBDR Protocol

BioFS and NBDR are complementary:

```
NBDR (biofile://)         BioFS (biofs://)
       │                        │
       │                        │
       ▼                        ▼
  S3 Storage              P2P Network
  (centralized)           (decentralized)
       │                        │
       └────────┬───────────────┘
                │
                ▼
         User's Genome Files
    (NFT-gated, consent-driven)
```

**When to use NBDR**:
- Centralized storage with GenoBank
- Fast access via presigned URLs
- Integration with existing services

**When to use BioFS**:
- Decentralized storage across nodes
- Local filesystem access
- Offline-first applications
- Community-hosted genomic databases

**Unified Access**:
```bash
# Same NFT, different protocols
biofile://story/0xC919.../1234/genome.vcf  → S3 (NBDR)
biofs://story/0xC919.../1234/genome.vcf    → P2P (BioFS)
```

---

## Security Model

### Threat Model

1. **Unauthorized Access**: Prevented by NFT ownership verification
2. **Data Leakage**: Content encrypted with NFT holder's key
3. **Consent Violation**: Gossip protocol ensures rapid unpinning
4. **Malicious Nodes**: Only authorized nodes can serve content
5. **MITM Attacks**: All communications over encrypted libp2p channels

### Cryptographic Primitives

```go
// Content encryption with NFT owner's public key
func encryptForNFT(content []byte, nftRef NFTReference) ([]byte, error) {
    ownerPubKey := getOwnerPublicKey(nftRef)
    return encrypt(content, ownerPubKey)
}

// Consent signature verification
func verifyConsentSignature(biocid BioCID, sig []byte) bool {
    message := biocid.String()
    ownerAddr := recoverAddress(message, sig)
    return isNFTOwner(biocid.NFTRef, ownerAddr)
}

// Deletion proof
func generateDeletionProof(biocid BioCID, deletedAt time.Time) []byte {
    proof := DeletionProof{
        BioCID:    biocid,
        DeletedAt: deletedAt,
        NodeID:    getLocalNodeID(),
        Signature: sign(biocid + deletedAt + nodeID),
    }
    return merkleTree.AddDeletion(proof)
}
```

---

## Performance Optimizations

### Content Chunking

```go
// Split large files into chunks for efficient P2P transfer
const CHUNK_SIZE = 256 * 1024 // 256 KB

func chunkContent(data []byte) []Chunk {
    chunks := []Chunk{}
    for i := 0; i < len(data); i += CHUNK_SIZE {
        end := i + CHUNK_SIZE
        if end > len(data) {
            end = len(data)
        }
        chunks = append(chunks, Chunk{
            Index: i / CHUNK_SIZE,
            Data:  data[i:end],
            Hash:  sha256(data[i:end]),
        })
    }
    return chunks
}
```

### Caching Strategy

```go
type BioFSCache struct {
    lru *lru.Cache
}

func (c *BioFSCache) Get(biocid BioCID) ([]byte, bool) {
    // Check consent before returning cached data
    if !checkConsent(biocid.NFTRef) {
        c.Evict(biocid)
        return nil, false
    }

    return c.lru.Get(biocid.String())
}
```

---

## Deployment Architecture

```
┌─────────────────────────────────────────────────────┐
│              GenoBank Infrastructure                 │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │   Seed     │  │   Seed     │  │   Seed     │   │
│  │   Node 1   │  │   Node 2   │  │   Node 3   │   │
│  │            │  │            │  │            │   │
│  │ biofs-     │  │ biofs-     │  │ biofs-     │   │
│  │ daemon     │  │ daemon     │  │ daemon     │   │
│  └────────────┘  └────────────┘  └────────────┘   │
│         │               │               │          │
└─────────┼───────────────┼───────────────┼──────────┘
          │               │               │
          └───────────────┴───────────────┘
                      │
          ┌───────────┴───────────┐
          │                       │
    ┌─────▼─────┐         ┌───────▼──────┐
    │  Research │         │   Hospital   │
    │  Labs     │         │   Nodes      │
    │  (biofs)  │         │   (biofs)    │
    └───────────┘         └──────────────┘
```

---

## Roadmap

### Phase 1: Core Protocol (Q1 2026)
- [ ] BioCID specification
- [ ] Consent smart contracts (Solidity)
- [ ] biofs-daemon (Go)
- [ ] DHT with consent awareness
- [ ] Gossip protocol for revocations

### Phase 2: FUSE Integration (Q2 2026)
- [ ] biofs-fuse driver
- [ ] Linux kernel module
- [ ] Mount/unmount commands
- [ ] File operations (read/write)
- [ ] Permission system

### Phase 3: P2P Network (Q3 2026)
- [ ] Libp2p integration
- [ ] Content chunking and transfer
- [ ] Node discovery
- [ ] Replication strategy
- [ ] Bandwidth optimization

### Phase 4: Production Deploy (Q4 2026)
- [ ] Seed nodes deployment
- [ ] CLI tools (biofs-cli)
- [ ] Desktop application (BioFS Manager)
- [ ] Integration with NBDR protocol
- [ ] Documentation and tutorials

---

## GDPR Compliance Matrix

| Article | Requirement | BioFS Implementation |
|---------|-------------|---------------------|
| **Article 17** | Right to Erasure | NFT burn → instant unpin across all nodes + cryptographic deletion proof |
| **Article 20** | Data Portability | NFT transfer → new owner gets immediate access, old owner loses access |
| **Article 25** | Data Protection by Design | NFT-gated access at protocol level, not application level |
| **Article 32** | Security of Processing | End-to-end encryption, consent verification, audit trails |

---

## License

MIT + Patent Grant (similar to IPFS)

---

## References

- IPFS Whitepaper: https://ipfs.io/ipfs/QmR7GSQM93Cx5eAg6a6yRzNde1FQv7uL6X1o4k7zrJa3LX
- libp2p Specifications: https://github.com/libp2p/specs
- FUSE Documentation: https://github.com/libfuse/libfuse
- ERC1155 Standard: https://eips.ethereum.org/EIPS/eip-1155
- GDPR Text: https://gdpr-info.eu/

---

**Status**: Ready for implementation
**Next Step**: Create repository structure and begin core daemon development