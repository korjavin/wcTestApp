package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// RelayServer represents a WebSocket relay server
type RelayServer struct {
	upgrader            websocket.Upgrader
	subscriptionManager *SubscriptionManager
	messageQueue        chan *Message
	clients             map[*websocket.Conn]string // connection -> clientID
	mutex               sync.RWMutex
	logger              Logger
}

// NewRelayServer creates a new relay server
func NewRelayServer(logger Logger) *RelayServer {
	return &RelayServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for educational purposes
			},
		},
		subscriptionManager: NewSubscriptionManager(logger),
		messageQueue:        make(chan *Message, 100),
		clients:             make(map[*websocket.Conn]string),
		logger:              logger,
	}
}

// Start starts the relay server
func (s *RelayServer) Start() {
	go s.processMessages()
}

// HandleWebSocket handles WebSocket connections
func (s *RelayServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to upgrade connection: %v", err))
		return
	}

	// Generate a client ID
	clientID := uuid.New().String()

	// Add the client to the clients map
	s.mutex.Lock()
	s.clients[conn] = clientID
	s.mutex.Unlock()

	s.logger.Info(fmt.Sprintf("Client %s connected", clientID))

	// Handle the connection
	go s.handleConnection(conn, clientID)
}

// handleConnection handles a WebSocket connection
func (s *RelayServer) handleConnection(conn *websocket.Conn, clientID string) {
	defer func() {
		// Unsubscribe from all topics
		s.subscriptionManager.UnsubscribeAll(clientID)

		// Remove the client from the clients map
		s.mutex.Lock()
		delete(s.clients, conn)
		s.mutex.Unlock()

		// Close the connection
		conn.Close()

		s.logger.Info(fmt.Sprintf("Client %s disconnected", clientID))
	}()

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to set read deadline: %v", err))
		return
	}

	// Set pong handler
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			s.logger.Error(fmt.Sprintf("Failed to set read deadline in pong handler: %v", err))
		}
		return nil
	})

	// Start ping ticker
	go s.pingClient(conn)

	// Read messages from the client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error(fmt.Sprintf("Unexpected close error: %v", err))
			}
			break
		}

		// Parse the JSON-RPC request
		request, err := ParseJSONRPCRequest(string(message))
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to parse JSON-RPC request: %v", err))
			s.sendErrorResponse(conn, 0, -32700, "Parse error")
			continue
		}

		// Handle the request
		s.handleRequest(conn, clientID, request)
	}
}

// pingClient sends ping messages to the client
func (s *RelayServer) pingClient(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
			s.logger.Error(fmt.Sprintf("Failed to send ping: %v", err))
			return
		}
	}
}

// handleRequest handles a JSON-RPC request
func (s *RelayServer) handleRequest(conn *websocket.Conn, clientID string, request *JSONRPCRequest) {
	switch request.Method {
	case "subscribe":
		s.handleSubscribe(conn, clientID, request)
	case "publish":
		s.handlePublish(conn, clientID, request)
	case "unsubscribe":
		s.handleUnsubscribe(conn, clientID, request)
	default:
		s.logger.Warn(fmt.Sprintf("Unknown method: %s", request.Method))
		s.sendErrorResponse(conn, request.ID, -32601, "Method not found")
	}
}

// handleSubscribe handles a subscribe request
func (s *RelayServer) handleSubscribe(conn *websocket.Conn, clientID string, request *JSONRPCRequest) {
	// Parse the parameters
	var params SubscribeParams
	paramsBytes, err := json.Marshal(request.Params)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to marshal params: %v", err))
		s.sendErrorResponse(conn, request.ID, -32602, "Invalid params")
		return
	}

	err = json.Unmarshal(paramsBytes, &params)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to unmarshal params: %v", err))
		s.sendErrorResponse(conn, request.ID, -32602, "Invalid params")
		return
	}

	// Subscribe to the topic
	err = s.subscriptionManager.Subscribe(params.Topic, clientID, conn)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to subscribe: %v", err))
		s.sendErrorResponse(conn, request.ID, -32000, "Subscription error")
		return
	}

	// Send a success response
	s.sendSuccessResponse(conn, request.ID, true)

	s.logger.Info(fmt.Sprintf("Client %s subscribed to topic %s", clientID, params.Topic))
}

// handlePublish handles a publish request
func (s *RelayServer) handlePublish(conn *websocket.Conn, clientID string, request *JSONRPCRequest) {
	// Parse the parameters
	var params PublishParams
	paramsBytes, err := json.Marshal(request.Params)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to marshal params: %v", err))
		s.sendErrorResponse(conn, request.ID, -32602, "Invalid params")
		return
	}

	err = json.Unmarshal(paramsBytes, &params)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to unmarshal params: %v", err))
		s.sendErrorResponse(conn, request.ID, -32602, "Invalid params")
		return
	}

	// Create a new message
	message := NewMessage(params.Topic, params.Message, params.TTL)

	// Add the message to the queue
	s.messageQueue <- message

	// Send a success response
	s.sendSuccessResponse(conn, request.ID, true)

	s.logger.Info(fmt.Sprintf("Client %s published message to topic %s", clientID, params.Topic))
}

// handleUnsubscribe handles an unsubscribe request
func (s *RelayServer) handleUnsubscribe(conn *websocket.Conn, clientID string, request *JSONRPCRequest) {
	// Parse the parameters
	var params UnsubscribeParams
	paramsBytes, err := json.Marshal(request.Params)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to marshal params: %v", err))
		s.sendErrorResponse(conn, request.ID, -32602, "Invalid params")
		return
	}

	err = json.Unmarshal(paramsBytes, &params)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to unmarshal params: %v", err))
		s.sendErrorResponse(conn, request.ID, -32602, "Invalid params")
		return
	}

	// Unsubscribe from the topic
	err = s.subscriptionManager.Unsubscribe(params.Topic, clientID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to unsubscribe: %v", err))
		s.sendErrorResponse(conn, request.ID, -32000, "Unsubscription error")
		return
	}

	// Send a success response
	s.sendSuccessResponse(conn, request.ID, true)

	s.logger.Info(fmt.Sprintf("Client %s unsubscribed from topic %s", clientID, params.Topic))
}

// processMessages processes messages in the queue
func (s *RelayServer) processMessages() {
	for message := range s.messageQueue {
		// Skip expired messages
		if message.IsExpired() {
			s.logger.Info(fmt.Sprintf("Skipping expired message for topic %s", message.Topic))
			continue
		}

		// Get subscribers for the topic
		subscribers := s.subscriptionManager.GetSubscribers(message.Topic)
		if len(subscribers) == 0 {
			s.logger.Info(fmt.Sprintf("No subscribers for topic %s", message.Topic))
			continue
		}

		// Create a JSON-RPC notification
		notification := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "message",
			"params": map[string]interface{}{
				"topic":   message.Topic,
				"message": message.Payload,
			},
		}

		// Marshal the notification
		notificationBytes, err := json.Marshal(notification)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to marshal notification: %v", err))
			continue
		}

		// Send the notification to all subscribers
		for _, subscriber := range subscribers {
			err := subscriber.Connection.WriteMessage(websocket.TextMessage, notificationBytes)
			if err != nil {
				s.logger.Error(fmt.Sprintf("Failed to send notification to client %s: %v", subscriber.ClientID, err))
				// Unsubscribe the client if we can't send messages
				s.subscriptionManager.UnsubscribeAll(subscriber.ClientID)
			}
		}

		s.logger.Info(fmt.Sprintf("Sent message to %d subscribers for topic %s", len(subscribers), message.Topic))
	}
}

// sendSuccessResponse sends a success response
func (s *RelayServer) sendSuccessResponse(conn *websocket.Conn, id int, result interface{}) {
	response := NewJSONRPCResponse(id, result)
	responseJSON, err := response.ToJSON()
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(responseJSON))
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to send response: %v", err))
	}
}

// sendErrorResponse sends an error response
func (s *RelayServer) sendErrorResponse(conn *websocket.Conn, id int, code int, message string) {
	response := NewJSONRPCErrorResponse(id, code, message)
	responseJSON, err := response.ToJSON()
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to marshal error response: %v", err))
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte(responseJSON))
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to send error response: %v", err))
	}
}

// GetStats returns statistics about the relay server
func (s *RelayServer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"clients":       s.subscriptionManager.GetClientCount(),
		"subscriptions": s.subscriptionManager.GetSubscriptionCount(),
		"topics":        s.subscriptionManager.GetTopicCount(),
	}
}
