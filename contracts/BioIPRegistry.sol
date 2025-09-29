// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

/**
 * @title BioIPRegistry
 * @dev Story Protocol PIL-compatible BioIP Asset registry for BioFS
 *
 * Integrates with Story Protocol's Programmable IP Licensing (PIL):
 * - Each BioIP Asset = Story Protocol IP Asset (IPA)
 * - Supports derivative creation with license tokens
 * - Tracks full lineage tree (parent → child → grandchild)
 * - GDPR-compliant consent-driven deletion
 * - Compatible with biofs:// URI scheme
 *
 * Derivative Flow (Story Protocol Compatible):
 * 1. Parent mints license tokens
 * 2. Child is minted WITHOUT license terms
 * 3. Child is registered as derivative using minted tokens
 * 4. License tokens are consumed (one-time use)
 */
contract BioIPRegistry is ERC1155, Ownable {
    using Counters for Counters.Counter;
    Counters.Counter private _tokenIds;

    enum ConsentState { PENDING, ACTIVE, REVOKED, DELETED }

    struct BioIPAsset {
        address owner;              // BioIP owner (data subject)
        uint256 tokenId;            // Token ID
        ConsentState consentState;  // Consent status
        uint256 createdAt;          // Creation timestamp
        uint256 revokedAt;          // Revocation timestamp
        bytes32 contentHash;        // Hash of genomic content
        string dataType;            // Type: vcf, bam, fastq, sqlite, csv
        uint256 dataSize;           // Size in bytes
        bytes32 bioCID;             // BioFS content identifier

        // Story Protocol Integration
        address ipAssetId;          // Story Protocol IP Asset address
        uint256 licenseTermsId;     // PIL license terms ID
        bool hasLicense;            // Whether this asset has license terms

        // Lineage tracking
        uint256 parentTokenId;      // Parent token (0 if root)
        uint256[] childTokenIds;    // Array of child tokens
        uint256 generation;         // 0=root, 1=child, 2=grandchild, etc.
        uint256 licenseTokenId;     // License token used to create this derivative
    }

    struct LicenseToken {
        uint256 tokenId;            // License token ID
        uint256 parentTokenId;      // Parent BioIP that minted this
        address mintedFor;          // Wallet this was minted for
        uint256 mintedAt;           // When it was minted
        bool consumed;              // Whether it's been used to create derivative
        uint256 consumedBy;         // Child token that consumed it (0 if not consumed)
    }

    struct DeletionProof {
        uint256 tokenId;
        uint256 deletedAt;
        bytes32 merkleRoot;
        uint256 nodeCount;
        address[] verifiers;
    }

    // Token ID => BioIP Asset
    mapping(uint256 => BioIPAsset) public bioips;

    // Token ID => Wallet => Has permission
    mapping(uint256 => mapping(address => bool)) public permittedWallets;

    // Token ID => Deletion proof
    mapping(uint256 => DeletionProof) public deletionProofs;

    // Wallet => Token IDs owned
    mapping(address => uint256[]) private ownerTokens;

    // License Token ID => License Token data
    mapping(uint256 => LicenseToken) public licenseTokens;

    // Counter for license tokens
    Counters.Counter private _licenseTokenIds;

    // Parent Token ID => Array of minted license token IDs
    mapping(uint256 => uint256[]) public parentLicenseTokens;

    // Events
    event BioIPMinted(
        uint256 indexed tokenId,
        address indexed owner,
        bytes32 contentHash,
        string dataType,
        bytes32 bioCID,
        address ipAssetId,
        uint256 licenseTermsId
    );

    event BioIPDerivativeCreated(
        uint256 indexed childTokenId,
        uint256 indexed parentTokenId,
        uint256 licenseTokenId,
        uint256 generation
    );

    event ConsentGranted(uint256 indexed tokenId, address indexed owner);
    event ConsentRevoked(uint256 indexed tokenId, address indexed owner, uint256 timestamp);
    event ContentDeleted(uint256 indexed tokenId, bytes32 merkleRoot, uint256 nodeCount);
    event PermissionGranted(uint256 indexed tokenId, address indexed wallet, address indexed grantor);
    event PermissionRevoked(uint256 indexed tokenId, address indexed wallet, address indexed revoker);
    event DeletionVerified(uint256 indexed tokenId, address indexed verifier, bytes32 nodeSignature);

    event LicenseTokenMinted(
        uint256 indexed licenseTokenId,
        uint256 indexed parentTokenId,
        address indexed mintedFor
    );

    event LicenseTokenConsumed(
        uint256 indexed licenseTokenId,
        uint256 indexed childTokenId
    );

    constructor() ERC1155("https://api.genobank.io/biofs/metadata/{id}") Ownable(msg.sender) {}

    /**
     * @dev Mint root BioIP Asset with license terms (becomes a parent)
     * @param contentHash Hash of the genomic content
     * @param dataType Type of data (vcf, bam, etc)
     * @param dataSize Size of data in bytes
     * @param bioCID BioFS content identifier
     * @param ipAssetId Story Protocol IP Asset address
     * @param licenseTermsId PIL license terms ID
     */
    function mintRootBioIP(
        bytes32 contentHash,
        string memory dataType,
        uint256 dataSize,
        bytes32 bioCID,
        address ipAssetId,
        uint256 licenseTermsId
    ) external returns (uint256) {
        _tokenIds.increment();
        uint256 newTokenId = _tokenIds.current();

        // Mint NFT
        _mint(msg.sender, newTokenId, 1, "");

        // Create BioIP Asset
        bioips[newTokenId] = BioIPAsset({
            owner: msg.sender,
            tokenId: newTokenId,
            consentState: ConsentState.ACTIVE,
            createdAt: block.timestamp,
            revokedAt: 0,
            contentHash: contentHash,
            dataType: dataType,
            dataSize: dataSize,
            bioCID: bioCID,
            ipAssetId: ipAssetId,
            licenseTermsId: licenseTermsId,
            hasLicense: true,              // Root has license
            parentTokenId: 0,              // No parent (root)
            childTokenIds: new uint256[](0),
            generation: 0,                 // Generation 0 (root)
            licenseTokenId: 0              // Not a derivative
        });

        ownerTokens[msg.sender].push(newTokenId);

        emit BioIPMinted(newTokenId, msg.sender, contentHash, dataType, bioCID, ipAssetId, licenseTermsId);

        return newTokenId;
    }

    /**
     * @dev Mint license tokens for creating derivatives
     * This must be done BEFORE creating the derivative
     * @param parentTokenId Parent BioIP token
     * @param receiver Wallet to receive license token
     * @param amount Number of license tokens to mint
     */
    function mintLicenseTokens(
        uint256 parentTokenId,
        address receiver,
        uint256 amount
    ) external returns (uint256[] memory) {
        require(balanceOf(msg.sender, parentTokenId) > 0, "Not BioIP owner");
        require(bioips[parentTokenId].hasLicense, "Parent has no license");
        require(bioips[parentTokenId].consentState == ConsentState.ACTIVE, "Consent not active");

        uint256[] memory newTokenIds = new uint256[](amount);

        for (uint256 i = 0; i < amount; i++) {
            _licenseTokenIds.increment();
            uint256 newLicenseTokenId = _licenseTokenIds.current();

            licenseTokens[newLicenseTokenId] = LicenseToken({
                tokenId: newLicenseTokenId,
                parentTokenId: parentTokenId,
                mintedFor: receiver,
                mintedAt: block.timestamp,
                consumed: false,
                consumedBy: 0
            });

            parentLicenseTokens[parentTokenId].push(newLicenseTokenId);
            newTokenIds[i] = newLicenseTokenId;

            emit LicenseTokenMinted(newLicenseTokenId, parentTokenId, receiver);
        }

        return newTokenIds;
    }

    /**
     * @dev Mint derivative BioIP WITHOUT license terms
     * @param contentHash Hash of the genomic content
     * @param dataType Type of data
     * @param dataSize Size of data in bytes
     * @param bioCID BioFS content identifier
     * @param ipAssetId Story Protocol IP Asset address (child)
     */
    function mintDerivativeBioIP(
        bytes32 contentHash,
        string memory dataType,
        uint256 dataSize,
        bytes32 bioCID,
        address ipAssetId
    ) external returns (uint256) {
        _tokenIds.increment();
        uint256 newTokenId = _tokenIds.current();

        // Mint NFT
        _mint(msg.sender, newTokenId, 1, "");

        // Create BioIP Asset WITHOUT license terms
        bioips[newTokenId] = BioIPAsset({
            owner: msg.sender,
            tokenId: newTokenId,
            consentState: ConsentState.ACTIVE,
            createdAt: block.timestamp,
            revokedAt: 0,
            contentHash: contentHash,
            dataType: dataType,
            dataSize: dataSize,
            bioCID: bioCID,
            ipAssetId: ipAssetId,
            licenseTermsId: 0,             // NO license terms yet
            hasLicense: false,             // Will inherit from parent
            parentTokenId: 0,              // Will be set in registerDerivative
            childTokenIds: new uint256[](0),
            generation: 0,                 // Will be set in registerDerivative
            licenseTokenId: 0              // Will be set in registerDerivative
        });

        ownerTokens[msg.sender].push(newTokenId);

        emit BioIPMinted(newTokenId, msg.sender, contentHash, dataType, bioCID, ipAssetId, 0);

        return newTokenId;
    }

    /**
     * @dev Register derivative relationship using license token
     * This consumes the license token (one-time use)
     * @param childTokenId Child BioIP token
     * @param licenseTokenId License token to consume
     */
    function registerDerivative(
        uint256 childTokenId,
        uint256 licenseTokenId
    ) external {
        require(balanceOf(msg.sender, childTokenId) > 0, "Not child owner");
        require(!licenseTokens[licenseTokenId].consumed, "License token already used");
        require(licenseTokens[licenseTokenId].mintedFor == msg.sender, "License not for you");

        uint256 parentTokenId = licenseTokens[licenseTokenId].parentTokenId;
        require(bioips[parentTokenId].consentState == ConsentState.ACTIVE, "Parent consent revoked");

        BioIPAsset storage parent = bioips[parentTokenId];
        BioIPAsset storage child = bioips[childTokenId];

        // Update child
        child.parentTokenId = parentTokenId;
        child.generation = parent.generation + 1;
        child.licenseTokenId = licenseTokenId;
        child.licenseTermsId = parent.licenseTermsId; // Inherit license

        // Update parent
        parent.childTokenIds.push(childTokenId);

        // Consume license token
        licenseTokens[licenseTokenId].consumed = true;
        licenseTokens[licenseTokenId].consumedBy = childTokenId;

        emit BioIPDerivativeCreated(childTokenId, parentTokenId, licenseTokenId, child.generation);
        emit LicenseTokenConsumed(licenseTokenId, childTokenId);
    }

    /**
     * @dev Grant consent for existing token (reactivate)
     */
    function grantConsent(uint256 tokenId) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not BioIP owner");
        require(bioips[tokenId].consentState == ConsentState.REVOKED, "Consent not revoked");

        bioips[tokenId].consentState = ConsentState.ACTIVE;
        bioips[tokenId].revokedAt = 0;

        emit ConsentGranted(tokenId, msg.sender);
    }

    /**
     * @dev Revoke consent - triggers unpinning across BioFS network
     */
    function revokeConsent(uint256 tokenId) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not BioIP owner");
        require(bioips[tokenId].consentState == ConsentState.ACTIVE, "Consent not active");

        bioips[tokenId].consentState = ConsentState.REVOKED;
        bioips[tokenId].revokedAt = block.timestamp;

        emit ConsentRevoked(tokenId, msg.sender, block.timestamp);
    }

    /**
     * @dev Burn NFT and delete content (GDPR Article 17)
     */
    function burnAndDelete(
        uint256 tokenId,
        bytes32 merkleRoot,
        uint256 nodeCount
    ) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not BioIP owner");

        // Burn NFT
        _burn(msg.sender, tokenId, 1);

        // Mark as deleted
        bioips[tokenId].consentState = ConsentState.DELETED;

        // Store deletion proof
        deletionProofs[tokenId] = DeletionProof({
            tokenId: tokenId,
            deletedAt: block.timestamp,
            merkleRoot: merkleRoot,
            nodeCount: nodeCount,
            verifiers: new address[](0)
        });

        emit ContentDeleted(tokenId, merkleRoot, nodeCount);
    }

    /**
     * @dev Check if wallet has consent to access data
     */
    function checkConsent(uint256 tokenId, address wallet) external view returns (bool) {
        BioIPAsset memory bioip = bioips[tokenId];

        if (bioip.consentState != ConsentState.ACTIVE) return false;
        if (bioip.owner == wallet) return true;
        if (permittedWallets[tokenId][wallet]) return true;

        return false;
    }

    /**
     * @dev Grant permission to another wallet
     */
    function grantPermission(uint256 tokenId, address wallet) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not BioIP owner");
        require(bioips[tokenId].consentState == ConsentState.ACTIVE, "Consent not active");

        permittedWallets[tokenId][wallet] = true;

        emit PermissionGranted(tokenId, wallet, msg.sender);
    }

    /**
     * @dev Revoke permission from wallet
     */
    function revokePermission(uint256 tokenId, address wallet) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not BioIP owner");

        permittedWallets[tokenId][wallet] = false;

        emit PermissionRevoked(tokenId, wallet, msg.sender);
    }

    /**
     * @dev Get lineage for a BioIP (all ancestors)
     */
    function getLineage(uint256 tokenId) external view returns (uint256[] memory) {
        uint256[] memory ancestors = new uint256[](bioips[tokenId].generation);
        uint256 currentToken = tokenId;

        for (uint256 i = 0; i < bioips[tokenId].generation; i++) {
            currentToken = bioips[currentToken].parentTokenId;
            ancestors[bioips[tokenId].generation - 1 - i] = currentToken;
        }

        return ancestors;
    }

    /**
     * @dev Get all descendants of a BioIP (children, grandchildren, etc)
     */
    function getDescendants(uint256 tokenId) external view returns (uint256[] memory) {
        return _getDescendantsRecursive(tokenId);
    }

    function _getDescendantsRecursive(uint256 tokenId) private view returns (uint256[] memory) {
        uint256[] memory children = bioips[tokenId].childTokenIds;

        if (children.length == 0) {
            return new uint256[](0);
        }

        // Count total descendants
        uint256 totalDescendants = children.length;
        for (uint256 i = 0; i < children.length; i++) {
            uint256[] memory grandchildren = _getDescendantsRecursive(children[i]);
            totalDescendants += grandchildren.length;
        }

        // Collect all descendants
        uint256[] memory allDescendants = new uint256[](totalDescendants);
        uint256 index = 0;

        for (uint256 i = 0; i < children.length; i++) {
            allDescendants[index++] = children[i];
            uint256[] memory grandchildren = _getDescendantsRecursive(children[i]);
            for (uint256 j = 0; j < grandchildren.length; j++) {
                allDescendants[index++] = grandchildren[j];
            }
        }

        return allDescendants;
    }

    /**
     * @dev Get license tokens available for a parent
     */
    function getAvailableLicenseTokens(uint256 parentTokenId) external view returns (uint256[] memory) {
        uint256[] memory allTokens = parentLicenseTokens[parentTokenId];
        uint256 availableCount = 0;

        // Count available tokens
        for (uint256 i = 0; i < allTokens.length; i++) {
            if (!licenseTokens[allTokens[i]].consumed) {
                availableCount++;
            }
        }

        // Collect available tokens
        uint256[] memory available = new uint256[](availableCount);
        uint256 index = 0;
        for (uint256 i = 0; i < allTokens.length; i++) {
            if (!licenseTokens[allTokens[i]].consumed) {
                available[index++] = allTokens[i];
            }
        }

        return available;
    }

    /**
     * @dev Get BioIP asset data
     */
    function getBioIP(uint256 tokenId) external view returns (BioIPAsset memory) {
        return bioips[tokenId];
    }

    /**
     * @dev Get license token data
     */
    function getLicenseToken(uint256 licenseTokenId) external view returns (LicenseToken memory) {
        return licenseTokens[licenseTokenId];
    }

    /**
     * @dev Get total number of minted BioIPs
     */
    function totalSupply() external view returns (uint256) {
        return _tokenIds.current();
    }

    /**
     * @dev Update base URI for token metadata
     */
    function setURI(string memory newuri) external onlyOwner {
        _setURI(newuri);
    }
}