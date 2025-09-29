# BioFS Story Protocol PIL Integration

## Overview

BioFS integrates with **Story Protocol's Programmable IP Licensing (PIL)** system to enable proper licensing and derivative tracking for genomic data. Each BioFS file becomes a **BioIP Asset** that maps to a Story Protocol **IP Asset (IPA)**.

## BioIP Assets vs IP Assets

| Concept | BioFS | Story Protocol |
|---------|-------|----------------|
| **Asset Type** | BioIP Asset | IP Asset (IPA) |
| **Registry** | BioIPRegistry.sol | IPAssetRegistry.sol |
| **Licensing** | PIL-compatible | PIL native |
| **Access Control** | NFT + Consent | NFT ownership |
| **Deletion** | GDPR-compliant | Permanent |

## Derivative Creation Flow

### ❌ Wrong Way (Error 0xb3e96921)

```solidity
// DON'T DO THIS - License token cannot be reused
parent_ip = mintRootBioIP(...)
license_token = mintLicenseToken(parent_ip, ...)

// Create first child - WORKS
child1 = mintDerivativeBioIP(...)
registerDerivative(child1, license_token)  // ✅ OK

// Try to create second child - FAILS
child2 = mintDerivativeBioIP(...)
registerDerivative(child2, license_token)  // ❌ 0xb3e96921 - Already consumed!
```

### ✅ Correct Way

```solidity
// Step 1: Create parent with license terms
parent_ip = mintRootBioIP(
    contentHash: hash(vcf_file),
    dataType: "vcf",
    ipAssetId: story_ip_asset,
    licenseTermsId: pil_terms
)

// Step 2: Mint FRESH license token for EACH derivative
license_token_1 = mintLicenseToken(parent_ip, wallet, 1)

// Step 3: Create child WITHOUT license terms
child_ip = mintDerivativeBioIP(
    contentHash: hash(sqlite_file),
    dataType: "sqlite",
    ipAssetId: child_story_ip_asset
    // NO license terms!
)

// Step 4: Register derivative with the minted token
registerDerivative(child_ip, license_token_1)
// License token is now consumed

// Step 5: For next derivative, mint NEW token
license_token_2 = mintLicenseToken(parent_ip, wallet, 1)
grandchild_ip = mintDerivativeBioIP(...)
registerDerivative(grandchild_ip, license_token_2)
```

## Complete Example: VCF → SQLite → CSV

### Scenario
1. User uploads VCF file (root)
2. System annotates with OpenCRAVAT → SQLite (child)
3. System generates CSV report → CSV (grandchild)

### Implementation

```go
package main

import (
    "context"
    "github.com/Genobank/biofs/pkg/bioip"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "math/big"
)

func CreateBioIPLineage(ctx context.Context) error {
    manager := bioip.NewBioIPManager()

    // Step 1: Upload VCF → Root BioIP with license
    vcfHash := [32]byte{/* SHA256 of VCF */}
    vcfBioCID := [32]byte{/* BioCID */}

    vcfTokenID, err := manager.MintRootBioIP(
        ctx,
        "story",
        vcfHash,
        "vcf",
        1024*1024*100, // 100 MB
        vcfBioCID,
        common.HexToAddress("0xStoryIPAsset..."),
        big.NewInt(1), // PIL license terms ID
        signer,
    )
    if err != nil {
        return err
    }

    // VCF is now: biofs://story/0xCollection.../1/vcf/genome.vcf

    // Step 2: Annotate VCF → SQLite (derivative)
    sqliteHash := [32]byte{/* SHA256 of SQLite */}
    sqliteBioCID := [32]byte{/* BioCID */}

    sqliteTokenID, err := manager.CreateDerivativeFlow(
        ctx,
        "story",
        vcfTokenID,              // Parent = VCF
        sqliteHash,
        "sqlite",
        1024*1024*50,           // 50 MB
        sqliteBioCID,
        common.HexToAddress("0xSQLiteIPAsset..."),
        signer,
    )
    if err != nil {
        return err
    }

    // SQLite is now: biofs://story/0xCollection.../2/sqlite/annotation.sqlite
    // Inherits PIL license from VCF parent

    // Step 3: Generate CSV → Grandchild
    csvHash := [32]byte{/* SHA256 of CSV */}
    csvBioCID := [32]byte{/* BioCID */}

    csvTokenID, err := manager.CreateDerivativeFlow(
        ctx,
        "story",
        sqliteTokenID,           // Parent = SQLite (which is child of VCF)
        csvHash,
        "csv",
        1024*10,                // 10 KB
        csvBioCID,
        common.HexToAddress("0xCSVIPAsset..."),
        signer,
    )
    if err != nil {
        return err
    }

    // CSV is now: biofs://story/0xCollection.../3/csv/report.csv
    // Inherits PIL license from SQLite parent (and ultimately from VCF grandparent)

    // Query lineage
    lineage, err := manager.GetLineage(ctx, "story", csvTokenID)
    // Returns: [VCF tokenId, SQLite tokenId]

    descendants, err := manager.GetDescendants(ctx, "story", vcfTokenID)
    // Returns: [SQLite tokenId, CSV tokenId]

    return nil
}
```

## Lineage Tree Structure

```
VCF (Generation 0)
├── SQLite (Generation 1)
│   ├── CSV Report (Generation 2)
│   └── PDF Report (Generation 2)
├── BAM Alignment (Generation 1)
│   └── Coverage Report (Generation 2)
└── AlphaGenome Analysis (Generation 1)
```

### On-Chain Data

```solidity
// VCF (Root)
BioIPAsset {
    tokenId: 1,
    hasLicense: true,
    licenseTermsId: 1,
    parentTokenId: 0,         // No parent
    generation: 0,
    childTokenIds: [2, 5, 6], // SQLite, BAM, AlphaGenome
    licenseTokenId: 0         // Not a derivative
}

// SQLite (Child of VCF)
BioIPAsset {
    tokenId: 2,
    hasLicense: false,        // Inherits from parent
    licenseTermsId: 1,        // Inherited from VCF
    parentTokenId: 1,         // VCF
    generation: 1,
    childTokenIds: [3, 4],    // CSV, PDF
    licenseTokenId: 27987     // License token used to create this
}

// CSV (Grandchild of VCF)
BioIPAsset {
    tokenId: 3,
    hasLicense: false,        // Inherits from parent
    licenseTermsId: 1,        // Inherited from VCF (via SQLite)
    parentTokenId: 2,         // SQLite
    generation: 2,
    childTokenIds: [],        // No children yet
    licenseTokenId: 27988     // Fresh license token from SQLite
}
```

## License Token Lifecycle

### Minting

```solidity
// Parent mints license tokens
uint256[] memory licenseTokens = mintLicenseTokens(
    parentTokenId: 1,    // VCF
    receiver: wallet,    // Who can use this token
    amount: 1            // How many tokens
)
// Returns: [27987]

// License token is now AVAILABLE
LicenseToken {
    tokenId: 27987,
    parentTokenId: 1,
    mintedFor: 0x742d35Cc...,
    consumed: false,
    consumedBy: 0
}
```

### Consumption

```solidity
// Register derivative consumes the token
registerDerivative(
    childTokenId: 2,       // SQLite
    licenseTokenId: 27987  // Fresh token from parent
)

// License token is now CONSUMED
LicenseToken {
    tokenId: 27987,
    parentTokenId: 1,
    mintedFor: 0x742d35Cc...,
    consumed: true,
    consumedBy: 2          // SQLite consumed it
}
```

### Reuse Attempt (Fails)

```solidity
// Try to use same token again
registerDerivative(
    childTokenId: 5,       // BAM
    licenseTokenId: 27987  // Already consumed!
)
// ❌ Reverts: "License token already used"
// Error: 0xb3e96921
```

## PIL License Terms

BioFS supports all PIL license types:

### Non-Commercial License

```javascript
const pilTerms = {
    transferable: true,
    commercialUse: false,      // No commercial use
    commercialAttribution: false,
    commercializerChecker: "0x0",
    commercializerCheckerData: "0x",
    commercialRevShare: 0,
    derivativesAllowed: true,  // Can create derivatives
    derivativesAttribution: true,
    derivativesApproval: false,
    derivativesReciprocal: false,
    territories: [],
    distributionChannels: [],
    contentRestrictions: []
}
```

### Commercial License with Royalties

```javascript
const pilTerms = {
    transferable: true,
    commercialUse: true,          // Commercial use allowed
    commercialAttribution: true,
    commercializerChecker: "0x0",
    commercializerCheckerData: "0x",
    commercialRevShare: 1500,     // 15% royalty
    derivativesAllowed: true,
    derivativesAttribution: true,
    derivativesApproval: false,
    derivativesReciprocal: true,  // Derivatives must use same license
    territories: [],
    distributionChannels: [],
    contentRestrictions: []
}
```

## Integration with GenoBank Services

### VCF Annotator

```python
# 1. User uploads VCF → Root BioIP
root_bioip = bioip_manager.mint_root_bioip(
    content_hash=hash(vcf_content),
    data_type="vcf",
    license_terms_id=1  # Non-commercial PIL
)

# 2. OpenCRAVAT annotates → SQLite derivative
sqlite_bioip = bioip_manager.create_derivative(
    parent_token_id=root_bioip['tokenId'],
    content_hash=hash(sqlite_content),
    data_type="sqlite"
)

# 3. Expert Curator generates report → CSV derivative
csv_bioip = bioip_manager.create_derivative(
    parent_token_id=sqlite_bioip['tokenId'],
    content_hash=hash(csv_content),
    data_type="csv"
)
```

### AlphaGenome

```python
# AlphaGenome analysis is a derivative of VCF
alphagenome_bioip = bioip_manager.create_derivative(
    parent_token_id=vcf_bioip['tokenId'],
    content_hash=hash(analysis_results),
    data_type="alphagenome"
)
```

### SOMOS Ancestry

```python
# Ancestry analysis is a derivative of 23andMe data
ancestry_bioip = bioip_manager.create_derivative(
    parent_token_id=dtc_data_bioip['tokenId'],
    content_hash=hash(ancestry_results),
    data_type="ancestry"
)
```

## NBDR Protocol Integration

BioFS PIL licensing works seamlessly with NBDR:

```
User uploads VCF
      │
      ▼
NBDR: biofile://story/0xC919.../1/vcf/genome.vcf
BioFS: biofs://story/0xC919.../1/vcf/genome.vcf
      │
      ├─ Has PIL license (non-commercial, derivatives allowed)
      │
      ▼
OpenCRAVAT annotates
      │
      ▼
NBDR: biofile://story/0xC919.../2/sqlite/annotation.sqlite
BioFS: biofs://story/0xC919.../2/sqlite/annotation.sqlite
      │
      ├─ Inherits PIL license from parent
      ├─ Generation 1 (child of VCF)
      │
      ▼
CSV report generated
      │
      ▼
NBDR: biofile://story/0xC919.../3/csv/report.csv
BioFS: biofs://story/0xC919.../3/csv/report.csv
      │
      └─ Inherits PIL license from parent
      └─ Generation 2 (grandchild of VCF)
```

## API Examples

### Check License Terms

```bash
# Get BioIP data
curl "https://genobank.app/api_nbdr/metadata?uri=biofs://story/0xC919.../2"

# Response includes:
{
    "tokenId": 2,
    "licenseTermsId": 1,
    "hasLicense": false,    # Inherits from parent
    "parentTokenId": 1,
    "generation": 1,
    "licenseTokenId": 27987
}
```

### Query Lineage

```bash
# Get full lineage tree
curl "https://genobank.app/api_bioip/lineage?token_id=3"

# Response:
{
    "token_id": 3,
    "data_type": "csv",
    "generation": 2,
    "ancestors": [
        {"token_id": 1, "data_type": "vcf", "generation": 0},
        {"token_id": 2, "data_type": "sqlite", "generation": 1}
    ],
    "descendants": []
}
```

## Best Practices

### 1. Always Mint Fresh License Tokens

```go
// ✅ CORRECT
for i := 0; i < numDerivatives; i++ {
    licenseToken := mintLicenseToken(parent, wallet, 1)
    child := mintDerivative(...)
    registerDerivative(child, licenseToken)
}

// ❌ WRONG
licenseToken := mintLicenseToken(parent, wallet, 1)
for i := 0; i < numDerivatives; i++ {
    child := mintDerivative(...)
    registerDerivative(child, licenseToken)  // Fails after first iteration
}
```

### 2. Use CreateDerivativeFlow Helper

```go
// ✅ RECOMMENDED - Handles everything automatically
childTokenID, err := manager.CreateDerivativeFlow(
    ctx, chain, parentTokenID,
    childHash, childType, childSize, childBioCID, childIPAsset,
    signer,
)
```

### 3. Track Lineage in MongoDB

```javascript
// Store lineage in MongoDB alongside blockchain data
{
    "token_id": 2,
    "biofile_uri": "biofs://story/0xC919.../2/sqlite/annotation.sqlite",
    "parent_token_id": 1,
    "generation": 1,
    "license_terms_id": 1,
    "license_token_used": 27987,
    "wallet_address": "0x742d35Cc...",
    "s3_path": "users/0x742d.../vcf_annotator/job123/annotation.sqlite"
}
```

## Troubleshooting

### Error: 0xb3e96921 - License token already used

**Cause**: Trying to reuse a consumed license token

**Solution**: Mint a fresh license token for each derivative

### Error: "Parent consent revoked"

**Cause**: Parent BioIP consent was revoked

**Solution**: Cannot create derivatives when parent consent is revoked

### Error: "License not for you"

**Cause**: Trying to use license token minted for different wallet

**Solution**: License tokens are wallet-specific, mint new one for your wallet

## Resources

- **Smart Contract**: `/contracts/BioIPRegistry.sol`
- **Go Package**: `/pkg/bioip/bioip.go`
- **Story Protocol Docs**: https://docs.story.foundation
- **PIL Terms**: https://docs.story.foundation/docs/pil-terms