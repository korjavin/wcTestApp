package wallet

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/korjavin/wctestapp/pkg/utils"
)

// SignRequest represents a request to sign a message
type SignRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

// SignResponse represents a response to a sign request
type SignResponse struct {
	ID     int    `json:"id"`
	Result string `json:"result"`
}

// NewPersonalSignRequest creates a new personal_sign request
func NewPersonalSignRequest(id int, message string, address string) *SignRequest {
	return &SignRequest{
		ID:     id,
		Method: "personal_sign",
		Params: []any{
			message,
			address,
		},
	}
}

// EncryptRequest encrypts a request for a session
func EncryptRequest(request *SignRequest, session *Session) (string, error) {
	// Marshal the request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Encrypt the request with the session's symmetric key
	encrypted, err := utils.EncryptWithSymmetricKey(requestJSON, session.SymKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt request: %w", err)
	}

	return encrypted, nil
}

// DecryptResponse decrypts a response from a session
func DecryptResponse(encryptedResponse string, session *Session) (*SignResponse, error) {
	// Decrypt the response with the session's symmetric key
	decrypted, err := utils.DecryptWithSymmetricKey(encryptedResponse, session.SymKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt response: %w", err)
	}

	// Unmarshal the response from JSON
	var response SignResponse
	err = json.Unmarshal(decrypted, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// VerifySignature verifies a signature
func VerifySignature(message string, signature string, address common.Address) (bool, error) {
	// Convert the message to bytes
	messageBytes := []byte(message)

	// Add the Ethereum signed message prefix
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageBytes), messageBytes)
	prefixedMessageBytes := []byte(prefixedMessage)

	// Hash the prefixed message
	hash := crypto.Keccak256Hash(prefixedMessageBytes)

	// Convert the signature from hex to bytes
	signatureBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// The signature should be 65 bytes: R (32 bytes) + S (32 bytes) + V (1 byte)
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(signatureBytes))
	}

	// Adjust the V value (last byte) for Ethereum's implementation
	if signatureBytes[64] < 27 {
		signatureBytes[64] += 27
	}

	// Recover the public key
	pubKey, err := crypto.SigToPub(hash.Bytes(), signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert the public key to an address
	recoveredAddress := crypto.PubkeyToAddress(*pubKey)

	// Compare the recovered address with the expected address
	return recoveredAddress == address, nil
}

// FormatSignature formats a signature for display
func FormatSignature(signature string) string {
	return signature
}

// GetMessageToSign gets the message to sign
func GetMessageToSign(message string) string {
	return message
}

// GetSignatureDetails gets the details of a signature
func GetSignatureDetails(message string, signature string) (map[string]string, error) {
	// Convert the signature from hex to bytes
	signatureBytes, err := hexutil.Decode(signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// The signature should be 65 bytes: R (32 bytes) + S (32 bytes) + V (1 byte)
	if len(signatureBytes) != 65 {
		return nil, fmt.Errorf("invalid signature length: %d", len(signatureBytes))
	}

	// Extract R, S, and V
	r := hexutil.Encode(signatureBytes[:32])
	s := hexutil.Encode(signatureBytes[32:64])
	v := signatureBytes[64]

	// Convert the message to bytes
	messageBytes := []byte(message)

	// Add the Ethereum signed message prefix
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageBytes), messageBytes)
	prefixedMessageBytes := []byte(prefixedMessage)

	// Hash the prefixed message
	hash := crypto.Keccak256Hash(prefixedMessageBytes)

	// Recover the public key
	pubKey, err := crypto.SigToPub(hash.Bytes(), signatureBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert the public key to an address
	recoveredAddress := crypto.PubkeyToAddress(*pubKey)

	return map[string]string{
		"message":           message,
		"signature":         signature,
		"r":                 r,
		"s":                 s,
		"v":                 fmt.Sprintf("0x%x", v),
		"recovered_address": recoveredAddress.Hex(),
		"message_hash":      hash.Hex(),
	}, nil
}

// GenerateKeyPair generates a new ECDSA key pair
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	return utils.GenerateKeyPair()
}
