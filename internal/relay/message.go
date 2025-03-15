package relay

import (
	"encoding/json"
	"fmt"
	"time"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	ID      int           `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SubscribeParams represents the parameters for a subscribe request
type SubscribeParams struct {
	Topic string `json:"topic"`
}

// PublishParams represents the parameters for a publish request
type PublishParams struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
}

// UnsubscribeParams represents the parameters for an unsubscribe request
type UnsubscribeParams struct {
	Topic string `json:"topic"`
}

// Message represents a message in the relay server
type Message struct {
	Topic     string    `json:"topic"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewMessage creates a new message
func NewMessage(topic string, payload string, ttl int) *Message {
	now := time.Now()
	return &Message{
		Topic:     topic,
		Payload:   payload,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(ttl) * time.Second),
	}
}

// IsExpired checks if the message is expired
func (m *Message) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}

// ToJSON converts the message to JSON
func (m *Message) ToJSON() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}
	return string(bytes), nil
}

// NewJSONRPCRequest creates a new JSON-RPC request
func NewJSONRPCRequest(id int, method string, params interface{}) *JSONRPCRequest {
	return &JSONRPCRequest{
		ID:      id,
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
}

// NewJSONRPCResponse creates a new JSON-RPC response
func NewJSONRPCResponse(id int, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		ID:      id,
		JSONRPC: "2.0",
		Result:  result,
	}
}

// NewJSONRPCErrorResponse creates a new JSON-RPC error response
func NewJSONRPCErrorResponse(id int, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		ID:      id,
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

// ParseJSONRPCRequest parses a JSON-RPC request from a string
func ParseJSONRPCRequest(data string) (*JSONRPCRequest, error) {
	var request JSONRPCRequest
	err := json.Unmarshal([]byte(data), &request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC request: %w", err)
	}
	return &request, nil
}

// ToJSON converts the JSON-RPC request to JSON
func (r *JSONRPCRequest) ToJSON() (string, error) {
	bytes, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}
	return string(bytes), nil
}

// ToJSON converts the JSON-RPC response to JSON
func (r *JSONRPCResponse) ToJSON() (string, error) {
	bytes, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-RPC response: %w", err)
	}
	return string(bytes), nil
}
