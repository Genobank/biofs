# BioFS - Biological Filesystem Protocol

**GDPR-Compliant Distributed Storage for Genomic Data**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![Solidity](https://img.shields.io/badge/Solidity-0.8.20-363636?logo=solidity)](https://soliditylang.org)

---

## 🧬 What is BioFS?

BioFS is a distributed filesystem protocol inspired by IPFS but designed specifically for genomic and medical data with **built-in GDPR/CCPA compliance** through NFT-gated access control and consent-driven deletion mechanisms.

```bash
# Mount your genome with NFT-based access control
sudo biofs mount biofs://story/0xC919.../1234 /mnt/my-genome

# Access files with automatic verification
cat /mnt/my-genome/genome.vcf
# ✅ NFT ownership verified
# ✅ Consent active
# → Content delivered
```

---

## 🎯 Key Features

| Feature | IPFS | BioFS |
|---------|------|-------|
| **Data Permanence** | Permanent | Consent-revocable |
| **Access Control** | Public | NFT-gated (ERC1155) |
| **Deletion** | Not possible | Cryptographically verified |
| **Mounting** | read-only | read/write with permissions |
| **Content ID** | CID (hash only) | BioCID (hash + NFT + consent) |
| **Use Case** | General data | Genomic/medical data |

---

## 🚀 Quick Start

### Installation

```bash
# Install BioFS daemon and CLI
go install github.com/Genobank/biofs/cmd/biofs@latest
go install github.com/Genobank/biofs/cmd/biofsctl@latest

# Initialize BioFS
biofs init

# Start daemon
biofs daemon
```

### Basic Usage

```bash
# Add genomic file with NFT reference
biofs add genome.vcf \
  --nft story/0xC91940118822D247B46d1eBA6B7Ed2A16F3aDC36/1234 \
  --wallet 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb4 \
  --sign 0xa5141ae...

# Output: biofs://story/0xC919.../1234/QmXoypiz.../genome.vcf

# Mount NFT content as filesystem
sudo biofs mount biofs://story/0xC919.../1234 /mnt/my-genome

# List files
ls /mnt/my-genome/
# genome.vcf
# annotation.sqlite
# report.pdf

# Access file (automatic NFT verification)
cat /mnt/my-genome/genome.vcf

# Grant permission to collaborator
biofs grant-permission \
  --nft story/0xC919.../1234 \
  --to 0x123... \
  --sign 0xa5141...

# Revoke consent (triggers global deletion)
biofs revoke-consent \
  --nft story/0xC919.../1234 \
  --sign 0xa5141...

# Verify deletion
biofs verify-deletion --nft story/0xC919.../1234
# ✅ Content deleted from 47/47 nodes
# ✅ Deletion proofs verified
# ✅ GDPR Article 17 compliant

# Unmount
sudo biofs unmount /mnt/my-genome
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────┐
│              Linux Kernel                       │
├─────────────────────────────────────────────────┤
│              FUSE Layer                         │
├─────────────────────────────────────────────────┤
│           biofs-fuse Driver                     │
│  • open()  → NFT ownership check                │
│  • read()  → Content retrieval                  │
│  • write() → Consent verification               │
│  • unlink() → Consent revocation                │
├─────────────────────────────────────────────────┤
│           biofs-daemon                          │
│  • Content Storage (pinning)                    │
│  • P2P Networking (libp2p)                      │
│  • NFT Verification (Story Protocol)            │
│  • Consent Monitoring (on-chain events)         │
│  • Cryptographic Deletion                       │
├─────────────────────────────────────────────────┤
│          Smart Contracts (ERC1155)              │
│  • ConsentRegistry.sol                          │
│  • grantConsent() / revokeConsent()             │
│  • burnAndDelete() / checkConsent()             │
└─────────────────────────────────────────────────┘
```

---

## 📦 BioCID Format

**Biological Content Identifier** extends IPFS CID with NFT and consent metadata:

```
biocid://v1/story/0xC919.../1234/QmXoypiz.../0xa5141ae...
         │   │     │        │     │          │
         │   │     │        │     │          └─ Consent signature
         │   │     │        │     └──────────── Content hash (SHA256)
         │   │     │        └────────────────── Token ID
         │   │     └─────────────────────────── Collection address
         │   └───────────────────────────────── Chain (story/avalanche/ethereum)
         └───────────────────────────────────── Protocol version
```

---

## 🔐 GDPR Compliance

### Right to Erasure (Article 17)

When an NFT owner revokes consent or burns the NFT:

1. **Smart Contract Event**: `ConsentRevoked` emitted on-chain
2. **Network Propagation**: All nodes receive event via gossip protocol
3. **Immediate Unpinning**: Content removed from all nodes
4. **Cryptographic Deletion**: Data overwritten with random bytes
5. **Deletion Proof**: Merkle proof generated showing absence
6. **Verifiable**: Anyone can verify deletion on-chain

```bash
# Revoke consent
biofs revoke-consent --nft story/0xC919.../1234 --sign 0x...

# Verify deletion across network
biofs verify-deletion --nft story/0xC919.../1234
# ✅ 47/47 nodes confirmed deletion
# ✅ Merkle proofs valid
# ✅ GDPR compliant
```

### Data Portability (Article 20)

NFT transfer = instant data migration:

```bash
# Transfer NFT to new wallet
biofsctl transfer-nft --nft story/0xC919.../1234 --to 0xNewOwner...

# New owner immediately gains access
# Old owner immediately loses access
```

---

## 🌐 Integration with NBDR Protocol

BioFS and NBDR (NFT-Based Data Routing) are complementary:

```
NBDR (biofile://)         BioFS (biofs://)
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

**Unified Access**:
```bash
# Same NFT, different protocols
biofile://story/0xC919.../1234/genome.vcf  → S3 (fast, centralized)
biofs://story/0xC919.../1234/genome.vcf    → P2P (decentralized, censorship-resistant)
```

---

## 🛠️ Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/Genobank/biofs.git
cd biofs

# Install dependencies
go mod download

# Build daemon
go build -o bin/biofs ./cmd/biofs

# Build CLI tools
go build -o bin/biofsctl ./cmd/biofsctl

# Run tests
go test ./...

# Deploy smart contracts
cd contracts
npm install
npx hardhat deploy --network story-testnet
```

### Project Structure

```
biofs/
├── cmd/
│   ├── biofs/          # Daemon + FUSE driver
│   └── biofsctl/       # CLI tool
├── pkg/
│   ├── biocid/         # BioCID implementation
│   ├── consent/        # Consent verification
│   ├── fuse/           # FUSE driver
│   ├── p2p/            # libp2p networking
│   ├── storage/        # Content storage
│   └── crypto/         # Cryptographic operations
├── contracts/
│   ├── ConsentRegistry.sol
│   ├── BioNFT.sol
│   └── test/
├── docs/
│   ├── ARCHITECTURE.md
│   ├── PROTOCOL.md
│   └── API.md
└── test/
```

---

## 📚 Documentation

- **[Architecture](docs/ARCHITECTURE.md)**: System design and components
- **[Protocol Specification](docs/PROTOCOL.md)**: BioCID format, P2P protocol
- **[API Reference](docs/API.md)**: CLI commands and Go API
- **[Smart Contracts](docs/CONTRACTS.md)**: ConsentRegistry and BioNFT
- **[GDPR Compliance](docs/GDPR.md)**: How BioFS ensures compliance

---

## 🔬 Use Cases

### Research Labs
```bash
# Share genomic datasets with collaborators
biofs add dataset.vcf --nft story/collection/123
biofs grant-permission --nft story/collection/123 --to 0xCollaborator...
```

### Hospitals
```bash
# Patient-controlled medical records
biofs mount biofs://story/medical-records/patient-id /mnt/records
# Patient can revoke access anytime
```

### Direct-to-Consumer Genomics
```bash
# Users own their 23andMe data
biofs add 23andme-data.txt --nft story/dtc-genomics/456
# Transfer NFT = transfer data ownership
```

### Precision Medicine
```bash
# Clinical trials with consent management
biofs add trial-data.csv --nft story/clinical-trial/789
# Auto-delete when trial ends
```

---

## 🛡️ Security

### Threat Model

1. **Unauthorized Access**: ✅ Prevented by NFT ownership verification
2. **Data Leakage**: ✅ Content encrypted with NFT holder's key
3. **Consent Violation**: ✅ Gossip protocol ensures rapid unpinning
4. **Malicious Nodes**: ✅ Only authorized nodes can serve content
5. **MITM Attacks**: ✅ All communications over encrypted libp2p channels

### Cryptographic Primitives

- **Content Hashing**: SHA256
- **Signatures**: ECDSA (secp256k1)
- **Encryption**: AES-256-GCM
- **Key Derivation**: HKDF-SHA256
- **Merkle Trees**: For deletion proofs

---

## 🗺️ Roadmap

### Phase 1: Core Protocol ✅ (Q1 2026)
- [x] BioCID specification
- [x] Architecture document
- [ ] Consent smart contracts
- [ ] biofs-daemon core

### Phase 2: FUSE Integration (Q2 2026)
- [ ] biofs-fuse driver
- [ ] Mount/unmount operations
- [ ] File I/O operations
- [ ] Permission system

### Phase 3: P2P Network (Q3 2026)
- [ ] libp2p integration
- [ ] DHT with consent awareness
- [ ] Gossip protocol
- [ ] Replication strategy

### Phase 4: Production (Q4 2026)
- [ ] Seed nodes deployment
- [ ] CLI tools release
- [ ] Desktop application
- [ ] NBDR integration

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Chat
- Discord: https://discord.gg/genobank
- Telegram: https://t.me/genobank

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details

---

## 🙏 Acknowledgments

- **IPFS Team**: For pioneering distributed filesystems
- **Protocol Labs**: For libp2p
- **Story Protocol**: For programmable IP infrastructure
- **GenoBank Community**: For genomic sovereignty vision

---

## 🔗 Links

- **Website**: https://genobank.io
- **NBDR Protocol**: https://genobank.io/nbdr-protocol.html
- **Docs**: https://docs.genobank.io
- **GitHub**: https://github.com/Genobank/biofs
- **Discord**: https://discord.gg/genobank

---

**Built with 💚 for Genomic Sovereignty**

*"Your genome, your data, your control"*