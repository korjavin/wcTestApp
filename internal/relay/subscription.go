package relay

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Subscription represents a subscription to a topic
type Subscription struct {
	Topic      string
	ClientID   string
	Connection *websocket.Conn
	CreatedAt  time.Time
}

// SubscriptionManager manages subscriptions to topics
type SubscriptionManager struct {
	subscriptions map[string][]*Subscription // topic -> subscriptions
	clients       map[string]*websocket.Conn // clientID -> connection
	mutex         sync.RWMutex
	logger        Logger
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(logger Logger) *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string][]*Subscription),
		clients:       make(map[string]*websocket.Conn),
		logger:        logger,
	}
}

// Subscribe subscribes a client to a topic
func (m *SubscriptionManager) Subscribe(topic string, clientID string, conn *websocket.Conn) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if the client is already subscribed to the topic
	for _, sub := range m.subscriptions[topic] {
		if sub.ClientID == clientID {
			m.logger.Info(fmt.Sprintf("Client %s is already subscribed to topic %s", clientID, topic))
			return nil
		}
	}

	// Create a new subscription
	subscription := &Subscription{
		Topic:      topic,
		ClientID:   clientID,
		Connection: conn,
		CreatedAt:  time.Now(),
	}

	// Add the subscription to the topic
	m.subscriptions[topic] = append(m.subscriptions[topic], subscription)

	// Add the client connection
	m.clients[clientID] = conn

	m.logger.Info(fmt.Sprintf("Client %s subscribed to topic %s", clientID, topic))
	return nil
}

// Unsubscribe unsubscribes a client from a topic
func (m *SubscriptionManager) Unsubscribe(topic string, clientID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Find the subscription
	subs, ok := m.subscriptions[topic]
	if !ok {
		m.logger.Warn(fmt.Sprintf("Topic %s not found for unsubscribe", topic))
		return nil
	}

	// Remove the subscription
	for i, sub := range subs {
		if sub.ClientID == clientID {
			m.subscriptions[topic] = append(subs[:i], subs[i+1:]...)
			m.logger.Info(fmt.Sprintf("Client %s unsubscribed from topic %s", clientID, topic))

			// If there are no more subscriptions for this topic, remove the topic
			if len(m.subscriptions[topic]) == 0 {
				delete(m.subscriptions, topic)
				m.logger.Info(fmt.Sprintf("Removed empty topic %s", topic))
			}

			return nil
		}
	}

	m.logger.Warn(fmt.Sprintf("Client %s not found in topic %s for unsubscribe", clientID, topic))
	return nil
}

// UnsubscribeAll unsubscribes a client from all topics
func (m *SubscriptionManager) UnsubscribeAll(clientID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Find all topics the client is subscribed to
	var topics []string
	for topic, subs := range m.subscriptions {
		for _, sub := range subs {
			if sub.ClientID == clientID {
				topics = append(topics, topic)
				break
			}
		}
	}

	// Unsubscribe from each topic
	for _, topic := range topics {
		for i, sub := range m.subscriptions[topic] {
			if sub.ClientID == clientID {
				m.subscriptions[topic] = append(m.subscriptions[topic][:i], m.subscriptions[topic][i+1:]...)
				m.logger.Info(fmt.Sprintf("Client %s unsubscribed from topic %s", clientID, topic))

				// If there are no more subscriptions for this topic, remove the topic
				if len(m.subscriptions[topic]) == 0 {
					delete(m.subscriptions, topic)
					m.logger.Info(fmt.Sprintf("Removed empty topic %s", topic))
				}

				break
			}
		}
	}

	// Remove the client connection
	delete(m.clients, clientID)
	m.logger.Info(fmt.Sprintf("Removed client %s", clientID))
}

// GetSubscribers returns all subscribers to a topic
func (m *SubscriptionManager) GetSubscribers(topic string) []*Subscription {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.subscriptions[topic]
}

// GetTopics returns all topics
func (m *SubscriptionManager) GetTopics() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	topics := make([]string, 0, len(m.subscriptions))
	for topic := range m.subscriptions {
		topics = append(topics, topic)
	}

	return topics
}

// GetClientConnection returns the connection for a client
func (m *SubscriptionManager) GetClientConnection(clientID string) *websocket.Conn {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.clients[clientID]
}

// GetClientCount returns the number of clients
func (m *SubscriptionManager) GetClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.clients)
}

// GetSubscriptionCount returns the number of subscriptions
func (m *SubscriptionManager) GetSubscriptionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	count := 0
	for _, subs := range m.subscriptions {
		count += len(subs)
	}

	return count
}

// GetTopicCount returns the number of topics
func (m *SubscriptionManager) GetTopicCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.subscriptions)
}

// Logger interface for logging
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}
