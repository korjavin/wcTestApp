package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GenerateKeyPair generates a new ECDSA key pair
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	return privateKey, publicKey, nil
}

// PrivateKeyToHex converts a private key to a hex string
func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	return hex.EncodeToString(crypto.FromECDSA(privateKey))
}

// PublicKeyToHex converts a public key to a hex string
func PublicKeyToHex(publicKey *ecdsa.PublicKey) string {
	return hex.EncodeToString(crypto.FromECDSAPub(publicKey))
}

// HexToPrivateKey converts a hex string to a private key
func HexToPrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	privateKey, err := crypto.ToECDSA(bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return privateKey, nil
}

// HexToPublicKey converts a hex string to a public key
func HexToPublicKey(hexKey string) (*ecdsa.PublicKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	publicKey, err := crypto.UnmarshalPubkey(bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	return publicKey, nil
}

// PublicKeyToAddress converts a public key to an Ethereum address
func PublicKeyToAddress(publicKey *ecdsa.PublicKey) common.Address {
	return crypto.PubkeyToAddress(*publicKey)
}

// GenerateRandomBytes generates random bytes of the specified length
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return bytes, nil
}

// GenerateSymmetricKey generates a random symmetric key
func GenerateSymmetricKey() (string, error) {
	key, err := GenerateRandomBytes(32) // AES-256 requires 32 bytes
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// EncryptWithSymmetricKey encrypts data using a symmetric key
func EncryptWithSymmetricKey(data []byte, keyStr string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return "", fmt.Errorf("invalid symmetric key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate a random IV
	iv, err := GenerateRandomBytes(aes.BlockSize)
	if err != nil {
		return "", err
	}

	// Create the GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nil, iv, data, nil)

	// Prepend the IV to the ciphertext
	result := append(iv, ciphertext...)

	// Encode the result as base64
	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptWithSymmetricKey decrypts data using a symmetric key
func DecryptWithSymmetricKey(encryptedStr string, keyStr string) ([]byte, error) {
	// Decode the base64 encrypted data
	encrypted, err := base64.StdEncoding.DecodeString(encryptedStr)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: %w", err)
	}

	// Decode the base64 key
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid symmetric key: %w", err)
	}

	// Create the cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create the GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract the IV from the encrypted data
	if len(encrypted) < aes.BlockSize {
		return nil, fmt.Errorf("encrypted data too short")
	}
	iv := encrypted[:aes.BlockSize]
	ciphertext := encrypted[aes.BlockSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// SignMessage signs a message with a private key
func SignMessage(message []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	// Hash the message using Keccak256
	hash := crypto.Keccak256Hash(message)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return signature, nil
}

// VerifySignature verifies a signature
func VerifySignature(message []byte, signature []byte, publicKey *ecdsa.PublicKey) bool {
	// Hash the message using Keccak256
	hash := crypto.Keccak256Hash(message)

	// Verify the signature
	return crypto.VerifySignature(crypto.FromECDSAPub(publicKey), hash.Bytes(), signature[:64])
}

// RecoverAddressFromSignature recovers the Ethereum address from a signature
func RecoverAddressFromSignature(message []byte, signature []byte) (common.Address, error) {
	// Hash the message using Keccak256
	hash := crypto.Keccak256Hash(message)

	// The signature should be 65 bytes: R (32 bytes) + S (32 bytes) + V (1 byte)
	if len(signature) != 65 {
		return common.Address{}, fmt.Errorf("invalid signature length: %d", len(signature))
	}

	// Recover the public key
	publicKey, err := crypto.Ecrecover(hash.Bytes(), signature)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert the public key to an Ethereum address
	pubCopy := make([]byte, len(publicKey))
	copy(pubCopy, publicKey)

	return common.BytesToAddress(crypto.Keccak256(pubCopy[1:])[12:]), nil
}

// GenerateRandomHex generates a random hex string of the specified length
func GenerateRandomHex(length int) (string, error) {
	bytes, err := GenerateRandomBytes(length / 2)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// GenerateRandomTopic generates a random topic for WalletConnect
func GenerateRandomTopic() (string, error) {
	return GenerateRandomHex(64)
}
