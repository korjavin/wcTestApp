package wallet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	// Log connection attempt with more details
	c.logger.Info(fmt.Sprintf("Connecting to relay server at %s for topic %s", c.relayURL, topic))
	c.logger.Debug(fmt.Sprintf("WebSocket connection details - URL: %s, Protocol: %s",
		c.relayURL, getWebSocketProtocol(c.relayURL)))
	c.logger.Info(fmt.Sprintf("NOTE: The wallet app may be using a different relay server than us"))
	c.logger.Info(fmt.Sprintf("Our relay server: %s", c.relayURL))

	// Connect to the relay server
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	// Add custom headers for debugging
	header := http.Header{}
	header.Add("X-Client-ID", "WalletClient")
	header.Add("X-Topic", topic)

	c.logger.Debug(fmt.Sprintf("Dialing WebSocket with headers: %v", header))

	conn, resp, err := dialer.Dial(c.relayURL, header)
	if err != nil {
		var statusCode int
		var responseBody string
		if resp != nil {
			statusCode = resp.StatusCode
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				responseBody = string(body)
				resp.Body.Close()
			}
		}

		c.logger.Error(fmt.Sprintf("Failed to connect to relay server: %v", err))
		c.logger.Debug(fmt.Sprintf("Connection failure details - Status: %d, Response: %s",
			statusCode, responseBody))
		return fmt.Errorf("failed to connect to relay server: %w (status: %d)", err, statusCode)
	}

	c.logger.Info(fmt.Sprintf("Successfully connected to relay server for topic %s", topic))
	c.logger.Debug(fmt.Sprintf("Connection established - Local: %s, Remote: %s",
		conn.LocalAddr().String(), conn.RemoteAddr().String()))

	// Subscribe to the topic
	subscribeRequest := relay.NewJSONRPCRequest(1, "subscribe", relay.SubscribeParams{
		Topic: topic,
	})

	subscribeRequestJSON, err := subscribeRequest.ToJSON()
	if err != nil {
		conn.Close()
		c.logger.Error(fmt.Sprintf("Failed to marshal subscribe request: %v", err))
		return fmt.Errorf("failed to marshal subscribe request: %w", err)
	}

	// Log the request being sent
	c.logger.Debug(fmt.Sprintf("Sending subscribe request: %s", subscribeRequestJSON))

	err = conn.WriteMessage(websocket.TextMessage, []byte(subscribeRequestJSON))
	if err != nil {
		conn.Close()
		c.logger.Error(fmt.Sprintf("Failed to send subscribe request: %v", err))
		return fmt.Errorf("failed to send subscribe request: %w", err)
	}

	// Read the response
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		c.logger.Error(fmt.Sprintf("Failed to read subscribe response: %v", err))
		return fmt.Errorf("failed to read subscribe response: %w", err)
	}

	// Log the raw response
	c.logger.Debug(fmt.Sprintf("Received raw subscribe response: %s", string(message)))

	// Parse the response
	var response relay.JSONRPCResponse
	err = json.Unmarshal(message, &response)
	if err != nil {
		conn.Close()
		c.logger.Error(fmt.Sprintf("Failed to parse subscribe response: %v", err))
		c.logger.Debug(fmt.Sprintf("Invalid JSON response: %s", string(message)))
		return fmt.Errorf("failed to parse subscribe response: %w", err)
	}

	// Check for errors
	if response.Error != nil {
		conn.Close()
		c.logger.Error(fmt.Sprintf("Subscribe error: %s (code: %d)",
			response.Error.Message, response.Error.Code))
		return fmt.Errorf("subscribe error: %s", response.Error.Message)
	}

	// Log successful subscription
	c.logger.Info(fmt.Sprintf("Successfully subscribed to topic: %s", topic))

	// Store the connection
	c.connections[topic] = conn

	// Start listening for messages
	go c.listenForMessages(topic, conn)

	return nil
}

// getWebSocketProtocol determines if the URL is using wss:// or ws:// based on the URL
func getWebSocketProtocol(url string) string {
	if strings.HasPrefix(url, "wss://") {
		return "wss"
	}
	return "ws"
}

// listenForMessages listens for messages on a topic
func (c *WalletClient) listenForMessages(topic string, conn *websocket.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	localAddr := conn.LocalAddr().String()

	c.logger.Info(fmt.Sprintf("Starting message listener for topic: %s", topic))
	c.logger.Debug(fmt.Sprintf("WebSocket connection details - Remote: %s, Local: %s, Protocol: %s",
		remoteAddr, localAddr, getWebSocketProtocol(c.relayURL)))

	defer func() {
		c.mutex.Lock()
		delete(c.connections, topic)
		c.mutex.Unlock()
		conn.Close()
		c.logger.Info(fmt.Sprintf("Disconnected from topic: %s", topic))
		c.logger.Debug(fmt.Sprintf("Closed WebSocket connection - Remote: %s, Local: %s",
			remoteAddr, localAddr))
	}()

	messageCount := 0

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.Info(fmt.Sprintf("WebSocket connection closed normally for topic %s: %v", topic, err))
			} else {
				c.logger.Error(fmt.Sprintf("Failed to read message from topic %s: %v", topic, err))
				c.logger.Debug(fmt.Sprintf("WebSocket error details - Remote: %s, Local: %s, Error: %v",
					remoteAddr, localAddr, err))
			}
			break
		}

		messageCount++
		c.logger.Debug(fmt.Sprintf("Received message #%d from topic %s (type: %d, size: %d bytes)",
			messageCount, topic, messageType, len(message)))

		// Log the raw message (truncated if too long)
		if len(message) > 1000 {
			c.logger.Debug(fmt.Sprintf("Raw message (truncated): %s...", string(message[:1000])))
		} else {
			c.logger.Debug(fmt.Sprintf("Raw message: %s", string(message)))
		}

		// Parse the message
		var notification struct {
			JSONRPC string `json:"jsonrpc"`
			Method  string `json:"method"`
			Params  struct {
				Topic   string `json:"topic"`
				Message string `json:"message"`
			} `json:"params"`
		}

		err = json.Unmarshal(message, &notification)
		if err != nil {
			c.logger.Error(fmt.Sprintf("Failed to parse notification from topic %s: %v", topic, err))
			c.logger.Debug(fmt.Sprintf("Invalid JSON message: %s", string(message)))
			continue
		}

		// Log the parsed notification
		c.logger.Debug(fmt.Sprintf("Parsed notification - Method: %s, Topic: %s, Message length: %d bytes",
			notification.Method, notification.Params.Topic, len(notification.Params.Message)))

		// Handle the message
		if notification.Method == "message" {
			c.logger.Info(fmt.Sprintf("Handling message from topic %s (message length: %d bytes)",
				notification.Params.Topic, len(notification.Params.Message)))
			c.handleMessage(notification.Params.Topic, notification.Params.Message)
		} else {
			c.logger.Info(fmt.Sprintf("Received notification with method: %s (not handling)", notification.Method))
		}
	}
}

// handleMessage handles a message from the relay server
func (c *WalletClient) handleMessage(topic string, encryptedMessage string) {
	c.logger.Info(fmt.Sprintf("Processing message from topic: %s (encrypted length: %d bytes)",
		topic, len(encryptedMessage)))

	// Log the first part of the encrypted message (for debugging)
	if len(encryptedMessage) > 100 {
		c.logger.Debug(fmt.Sprintf("Encrypted message (first 100 chars): %s...", encryptedMessage[:100]))
	} else {
		c.logger.Debug(fmt.Sprintf("Encrypted message: %s", encryptedMessage))
	}

	// Find the session for this topic
	var session *Session
	var sessionSource string

	if session = c.sessionManager.GetSessionByPairingTopic(topic); session != nil {
		sessionSource = "pairing topic"
	} else if session = c.sessionManager.GetSessionBySessionTopic(topic); session != nil {
		sessionSource = "session topic"
	} else {
		c.logger.Warn(fmt.Sprintf("No session found for topic: %s", topic))
		c.logger.Debug(fmt.Sprintf("Active sessions: %d", len(c.sessionManager.GetActiveSessions())))
		return
	}

	c.logger.Debug(fmt.Sprintf("Found session %s via %s (status: %s)",
		session.ID, sessionSource, session.Status))

	// Decrypt the message
	startTime := time.Now()
	decrypted, err := c.decryptMessage(encryptedMessage, session)
	decryptDuration := time.Since(startTime)

	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to decrypt message: %v", err))
		c.logger.Debug(fmt.Sprintf("Decryption failure details - Session: %s, Error: %v",
			session.ID, err))
		return
	}

	c.logger.Info(fmt.Sprintf("Successfully decrypted message in %s", decryptDuration))

	// Log the decrypted message (truncated if too long)
	if len(decrypted) > 500 {
		c.logger.Debug(fmt.Sprintf("Decrypted message (truncated): %s...", decrypted[:500]))
	} else {
		c.logger.Debug(fmt.Sprintf("Decrypted message: %s", decrypted))
	}

	// Try to parse the decrypted message as JSON for better logging
	var jsonMessage map[string]interface{}
	if err := json.Unmarshal([]byte(decrypted), &jsonMessage); err == nil {
		prettyJSON, _ := json.MarshalIndent(jsonMessage, "", "  ")
		c.logger.Debug(fmt.Sprintf("Parsed JSON message: %s", string(prettyJSON)))

		// Log specific message types
		if method, ok := jsonMessage["method"].(string); ok {
			c.logger.Info(fmt.Sprintf("Message method: %s", method))
		}
	}

	// TODO: Handle the decrypted message based on its type
	c.logger.Info(fmt.Sprintf("Message handling completed for topic: %s", topic))
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
