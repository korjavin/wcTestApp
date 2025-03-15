package wallet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/korjavin/wctestapp/internal/relay"
)

// WalletClient represents a WalletConnect client
type WalletClient struct {
	sessionManager *SessionManager
	relayURL       string
	connections    map[string]*websocket.Conn // topic -> connection
	mutex          sync.RWMutex
	logger         Logger
}

// Logger interface for logging
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

// NewWalletClient creates a new WalletConnect client
func NewWalletClient(relayURL string, logger Logger) *WalletClient {
	return &WalletClient{
		sessionManager: NewSessionManager(),
		relayURL:       relayURL,
		connections:    make(map[string]*websocket.Conn),
		logger:         logger,
	}
}

// CreateSession creates a new WalletConnect session
func (c *WalletClient) CreateSession() (*Session, error) {
	c.logger.Info("Creating new WalletConnect session")

	// Create a new session
	session, err := c.sessionManager.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	c.logger.Info(fmt.Sprintf("Created session with ID: %s", session.ID))

	return session, nil
}

// ConnectToRelay connects to the relay server for a session
func (c *WalletClient) ConnectToRelay(session *Session) error {
	c.logger.Info(fmt.Sprintf("Connecting to relay server for session: %s", session.ID))

	// Connect to the relay server for the pairing topic
	err := c.connectToTopic(session.PairingTopic)
	if err != nil {
		return fmt.Errorf("failed to connect to pairing topic: %w", err)
	}

	c.logger.Info(fmt.Sprintf("Connected to pairing topic: %s", session.PairingTopic))

	return nil
}

// connectToTopic connects to a topic on the relay server
func (c *WalletClient) connectToTopic(topic string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we're already connected to this topic
	if _, ok := c.connections[topic]; ok {
		c.logger.Info(fmt.Sprintf("Already connected to topic: %s", topic))
		return nil
	}

	// Connect to the relay server
	conn, _, err := websocket.DefaultDialer.Dial(c.relayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay server: %w", err)
	}

	// Subscribe to the topic
	subscribeRequest := relay.NewJSONRPCRequest(1, "subscribe", relay.SubscribeParams{
		Topic: topic,
	})

	subscribeRequestJSON, err := subscribeRequest.ToJSON()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to marshal subscribe request: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(subscribeRequestJSON))
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to send subscribe request: %w", err)
	}

	// Read the response
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to read subscribe response: %w", err)
	}

	// Parse the response
	var response relay.JSONRPCResponse
	err = json.Unmarshal(message, &response)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to parse subscribe response: %w", err)
	}

	// Check for errors
	if response.Error != nil {
		conn.Close()
		return fmt.Errorf("subscribe error: %s", response.Error.Message)
	}

	// Store the connection
	c.connections[topic] = conn

	// Start listening for messages
	go c.listenForMessages(topic, conn)

	return nil
}

// listenForMessages listens for messages on a topic
func (c *WalletClient) listenForMessages(topic string, conn *websocket.Conn) {
	defer func() {
		c.mutex.Lock()
		delete(c.connections, topic)
		c.mutex.Unlock()
		conn.Close()
		c.logger.Info(fmt.Sprintf("Disconnected from topic: %s", topic))
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to read message: %v", err))
			break
		}

		// Parse the message
		var notification struct {
			Method string `json:"method"`
			Params struct {
				Topic   string `json:"topic"`
				Message string `json:"message"`
			} `json:"params"`
		}

		err = json.Unmarshal(message, &notification)
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to parse notification: %v", err))
			continue
		}

		// Handle the message
		if notification.Method == "message" {
			c.handleMessage(notification.Params.Topic, notification.Params.Message)
		}
	}
}

// handleMessage handles a message from the relay server
func (c *WalletClient) handleMessage(topic string, encryptedMessage string) {
	c.logger.Info(fmt.Sprintf("Received message on topic: %s", topic))

	// Find the session for this topic
	var session *Session
	if session = c.sessionManager.GetSessionByPairingTopic(topic); session == nil {
		if session = c.sessionManager.GetSessionBySessionTopic(topic); session == nil {
			c.logger.Warn(fmt.Sprintf("No session found for topic: %s", topic))
			return
		}
	}

	// Decrypt the message
	decrypted, err := c.decryptMessage(encryptedMessage, session)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to decrypt message: %v", err))
		return
	}

	c.logger.Info(fmt.Sprintf("Decrypted message: %s", decrypted))

	// TODO: Handle the decrypted message based on its type
}

// decryptMessage decrypts a message for a session
func (c *WalletClient) decryptMessage(encryptedMessage string, session *Session) (string, error) {
	// Decrypt the message with the session's symmetric key
	decrypted, err := DecryptResponse(encryptedMessage, session)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt message: %w", err)
	}

	return fmt.Sprintf("%+v", decrypted), nil
}

// SignMessage requests a signature for a message
func (c *WalletClient) SignMessage(session *Session, message string) (string, error) {
	c.logger.Info(fmt.Sprintf("Requesting signature for message: %s", message))

	// Check if the session is active
	if session.Status != SessionStatusActive {
		return "", fmt.Errorf("session is not active")
	}

	// Create a sign request
	request := NewPersonalSignRequest(1, message, session.WalletAddress.Hex())

	// Encrypt the request
	encrypted, err := EncryptRequest(request, session)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt request: %w", err)
	}

	// Connect to the session topic if not already connected
	err = c.connectToTopic(session.SessionTopic)
	if err != nil {
		return "", fmt.Errorf("failed to connect to session topic: %w", err)
	}

	// Send the request
	c.mutex.RLock()
	conn := c.connections[session.SessionTopic]
	c.mutex.RUnlock()

	if conn == nil {
		return "", fmt.Errorf("not connected to session topic")
	}

	// Create a publish request
	publishRequest := relay.NewJSONRPCRequest(2, "publish", relay.PublishParams{
		Topic:   session.SessionTopic,
		Message: encrypted,
		TTL:     300, // 5 minutes
	})

	publishRequestJSON, err := publishRequest.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal publish request: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(publishRequestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to send publish request: %w", err)
	}

	c.logger.Info("Sent sign request to wallet")

	// TODO: Wait for the response
	// For now, we'll just return a placeholder
	return "Signature request sent. Waiting for wallet approval...", nil
}

// GetActiveSessions gets all active sessions
func (c *WalletClient) GetActiveSessions() []*Session {
	return c.sessionManager.GetActiveSessions()
}

// GetSession gets a session by ID
func (c *WalletClient) GetSession(id string) *Session {
	return c.sessionManager.GetSession(id)
}

// DisconnectSession disconnects a session
func (c *WalletClient) DisconnectSession(session *Session) error {
	c.logger.Info(fmt.Sprintf("Disconnecting session: %s", session.ID))

	// Disconnect from the pairing topic
	c.mutex.Lock()
	if conn, ok := c.connections[session.PairingTopic]; ok {
		conn.Close()
		delete(c.connections, session.PairingTopic)
	}

	// Disconnect from the session topic
	if conn, ok := c.connections[session.SessionTopic]; ok {
		conn.Close()
		delete(c.connections, session.SessionTopic)
	}
	c.mutex.Unlock()

	// Update the session status
	session.Disconnect()

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (c *WalletClient) CleanupExpiredSessions() {
	c.sessionManager.CleanupExpiredSessions()
}

// SetWalletAddress sets the wallet address for a session
func (c *WalletClient) SetWalletAddress(session *Session, address common.Address) {
	session.SetWalletAddress(address)
}

// GetWalletAddress gets the wallet address for a session
func (c *WalletClient) GetWalletAddress(session *Session) common.Address {
	return session.WalletAddress
}

// GetSignatureDetails gets the details of a signature
func (c *WalletClient) GetSignatureDetails(message, signature string) (map[string]string, error) {
	return GetSignatureDetails(message, signature)
}

// StartCleanupTask starts a task to periodically clean up expired sessions
func (c *WalletClient) StartCleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			c.CleanupExpiredSessions()
		}
	}()
}
