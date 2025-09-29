// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

/**
 * @title ConsentRegistry
 * @dev ERC1155 NFT contract with consent management for BioFS
 *
 * Key features:
 * - NFT-gated access to genomic data
 * - Consent granting and revocation
 * - GDPR Article 17 compliance (right to erasure)
 * - Delegated permissions for collaborators
 * - On-chain deletion proofs
 */
contract ConsentRegistry is ERC1155, Ownable {
    using Counters for Counters.Counter;
    Counters.Counter private _tokenIds;

    enum ConsentState { PENDING, ACTIVE, REVOKED, DELETED }

    struct ConsentMetadata {
        address owner;              // NFT owner (data subject)
        uint256 tokenId;            // NFT token ID
        ConsentState state;         // Current consent state
        uint256 createdAt;          // Timestamp of consent grant
        uint256 revokedAt;          // Timestamp of revocation (0 if not revoked)
        bytes32 contentHash;        // Hash of genomic content
        string dataType;            // Type of data (vcf, bam, fastq, etc)
        uint256 dataSize;           // Size in bytes
        bytes32 bioCID;             // BioFS content identifier
    }

    struct DeletionProof {
        uint256 tokenId;            // Token that was deleted
        uint256 deletedAt;          // Timestamp of deletion
        bytes32 merkleRoot;         // Merkle root proving deletion across nodes
        uint256 nodeCount;          // Number of nodes that deleted
        address[] verifiers;        // Addresses of nodes that verified deletion
    }

    // Token ID => Consent metadata
    mapping(uint256 => ConsentMetadata) public consents;

    // Token ID => Wallet => Has permission
    mapping(uint256 => mapping(address => bool)) public permittedWallets;

    // Token ID => Deletion proof
    mapping(uint256 => DeletionProof) public deletionProofs;

    // Wallet => Token IDs owned
    mapping(address => uint256[]) private ownerTokens;

    // Events
    event ConsentGranted(
        uint256 indexed tokenId,
        address indexed owner,
        bytes32 contentHash,
        string dataType,
        bytes32 bioCID
    );

    event ConsentRevoked(
        uint256 indexed tokenId,
        address indexed owner,
        uint256 timestamp
    );

    event ContentDeleted(
        uint256 indexed tokenId,
        bytes32 merkleRoot,
        uint256 nodeCount
    );

    event PermissionGranted(
        uint256 indexed tokenId,
        address indexed wallet,
        address indexed grantor
    );

    event PermissionRevoked(
        uint256 indexed tokenId,
        address indexed wallet,
        address indexed revoker
    );

    event DeletionVerified(
        uint256 indexed tokenId,
        address indexed verifier,
        bytes32 nodeSignature
    );

    constructor() ERC1155("https://api.genobank.io/biofs/metadata/{id}") Ownable(msg.sender) {}

    /**
     * @dev Mint new NFT and grant consent for genomic data
     * @param contentHash Hash of the genomic content
     * @param dataType Type of data (vcf, bam, etc)
     * @param dataSize Size of data in bytes
     * @param bioCID BioFS content identifier
     */
    function mintAndGrantConsent(
        bytes32 contentHash,
        string memory dataType,
        uint256 dataSize,
        bytes32 bioCID
    ) external returns (uint256) {
        _tokenIds.increment();
        uint256 newTokenId = _tokenIds.current();

        // Mint NFT to sender
        _mint(msg.sender, newTokenId, 1, "");

        // Create consent metadata
        consents[newTokenId] = ConsentMetadata({
            owner: msg.sender,
            tokenId: newTokenId,
            state: ConsentState.ACTIVE,
            createdAt: block.timestamp,
            revokedAt: 0,
            contentHash: contentHash,
            dataType: dataType,
            dataSize: dataSize,
            bioCID: bioCID
        });

        // Track owner's tokens
        ownerTokens[msg.sender].push(newTokenId);

        emit ConsentGranted(newTokenId, msg.sender, contentHash, dataType, bioCID);

        return newTokenId;
    }

    /**
     * @dev Grant consent for existing token (reactivate)
     * @param tokenId Token to grant consent for
     */
    function grantConsent(uint256 tokenId) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");
        require(consents[tokenId].state == ConsentState.REVOKED, "Consent not revoked");

        consents[tokenId].state = ConsentState.ACTIVE;
        consents[tokenId].revokedAt = 0;

        emit ConsentGranted(
            tokenId,
            msg.sender,
            consents[tokenId].contentHash,
            consents[tokenId].dataType,
            consents[tokenId].bioCID
        );
    }

    /**
     * @dev Revoke consent - triggers unpinning across BioFS network
     * @param tokenId Token to revoke consent for
     */
    function revokeConsent(uint256 tokenId) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");
        require(consents[tokenId].state == ConsentState.ACTIVE, "Consent not active");

        consents[tokenId].state = ConsentState.REVOKED;
        consents[tokenId].revokedAt = block.timestamp;

        emit ConsentRevoked(tokenId, msg.sender, block.timestamp);
    }

    /**
     * @dev Burn NFT and delete content (GDPR Article 17)
     * @param tokenId Token to burn and delete
     * @param merkleRoot Merkle root proving deletion across nodes
     * @param nodeCount Number of nodes that deleted content
     */
    function burnAndDelete(
        uint256 tokenId,
        bytes32 merkleRoot,
        uint256 nodeCount
    ) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");

        // Burn NFT
        _burn(msg.sender, tokenId, 1);

        // Mark as deleted
        consents[tokenId].state = ConsentState.DELETED;

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
     * @dev Verify deletion from a BioFS node
     * @param tokenId Token that was deleted
     * @param nodeSignature Signature proving deletion
     */
    function verifyDeletion(uint256 tokenId, bytes32 nodeSignature) external {
        require(consents[tokenId].state == ConsentState.DELETED, "Not deleted");

        // Add verifier to proof
        deletionProofs[tokenId].verifiers.push(msg.sender);

        emit DeletionVerified(tokenId, msg.sender, nodeSignature);
    }

    /**
     * @dev Check if wallet has consent to access data
     * @param tokenId Token to check
     * @param wallet Wallet address to check
     * @return hasConsent True if wallet has active consent
     */
    function checkConsent(uint256 tokenId, address wallet) external view returns (bool hasConsent) {
        ConsentMetadata memory consent = consents[tokenId];

        // Consent must be active
        if (consent.state != ConsentState.ACTIVE) return false;

        // Owner always has consent
        if (consent.owner == wallet) return true;

        // Check delegated permissions
        if (permittedWallets[tokenId][wallet]) return true;

        return false;
    }

    /**
     * @dev Grant permission to another wallet (collaborator)
     * @param tokenId Token to grant permission for
     * @param wallet Wallet to grant permission to
     */
    function grantPermission(uint256 tokenId, address wallet) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");
        require(consents[tokenId].state == ConsentState.ACTIVE, "Consent not active");
        require(wallet != address(0), "Invalid wallet");

        permittedWallets[tokenId][wallet] = true;

        emit PermissionGranted(tokenId, wallet, msg.sender);
    }

    /**
     * @dev Revoke permission from wallet
     * @param tokenId Token to revoke permission for
     * @param wallet Wallet to revoke permission from
     */
    function revokePermission(uint256 tokenId, address wallet) external {
        require(balanceOf(msg.sender, tokenId) > 0, "Not NFT owner");

        permittedWallets[tokenId][wallet] = false;

        emit PermissionRevoked(tokenId, wallet, msg.sender);
    }

    /**
     * @dev Get all tokens owned by an address
     * @param owner Address to query
     * @return tokenIds Array of token IDs
     */
    function getOwnerTokens(address owner) external view returns (uint256[] memory) {
        return ownerTokens[owner];
    }

    /**
     * @dev Get consent metadata for a token
     * @param tokenId Token to query
     * @return metadata Consent metadata
     */
    function getConsentMetadata(uint256 tokenId) external view returns (ConsentMetadata memory) {
        return consents[tokenId];
    }

    /**
     * @dev Get deletion proof for a token
     * @param tokenId Token to query
     * @return proof Deletion proof
     */
    function getDeletionProof(uint256 tokenId) external view returns (DeletionProof memory) {
        require(consents[tokenId].state == ConsentState.DELETED, "Not deleted");
        return deletionProofs[tokenId];
    }

    /**
     * @dev Check if content has been deleted
     * @param tokenId Token to check
     * @return isDeleted True if deleted
     * @return nodeCount Number of nodes that verified deletion
     */
    function isDeleted(uint256 tokenId) external view returns (bool isDeleted, uint256 nodeCount) {
        if (consents[tokenId].state == ConsentState.DELETED) {
            return (true, deletionProofs[tokenId].nodeCount);
        }
        return (false, 0);
    }

    /**
     * @dev Get total number of minted tokens
     * @return count Total token count
     */
    function totalSupply() external view returns (uint256) {
        return _tokenIds.current();
    }

    /**
     * @dev Update base URI for token metadata
     * @param newuri New base URI
     */
    function setURI(string memory newuri) external onlyOwner {
        _setURI(newuri);
    }
}