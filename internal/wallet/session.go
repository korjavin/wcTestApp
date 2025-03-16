package wallet

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/korjavin/wctestapp/pkg/utils"
)

// SessionStatus represents the status of a WalletConnect session
type SessionStatus string

const (
	// SessionStatusPending indicates a pending session
	SessionStatusPending SessionStatus = "pending"
	// SessionStatusActive indicates an active session
	SessionStatusActive SessionStatus = "active"
	// SessionStatusDisconnected indicates a disconnected session
	SessionStatusDisconnected SessionStatus = "disconnected"
)

// Session represents a WalletConnect session
type Session struct {
	ID            string            `json:"id"`
	PairingTopic  string            `json:"pairing_topic"`
	SessionTopic  string            `json:"session_topic"`
	SymKey        string            `json:"sym_key"`
	ClientID      string            `json:"client_id"`
	PeerID        string            `json:"peer_id"`
	ClientPubKey  *ecdsa.PublicKey  `json:"-"`
	ClientPrivKey *ecdsa.PrivateKey `json:"-"`
	PeerPubKey    *ecdsa.PublicKey  `json:"-"`
	WalletAddress common.Address    `json:"wallet_address"`
	Status        SessionStatus     `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	ExpiresAt     time.Time         `json:"expires_at"`
}

// NewSession creates a new WalletConnect session
func NewSession() (*Session, error) {
	// Generate a random session ID
	id, err := utils.GenerateRandomHex(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate a random pairing topic
	pairingTopic, err := utils.GenerateRandomTopic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pairing topic: %w", err)
	}

	// Generate a random session topic
	sessionTopic, err := utils.GenerateRandomTopic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session topic: %w", err)
	}

	// Generate a symmetric key
	symKey, err := utils.GenerateSymmetricKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate symmetric key: %w", err)
	}

	// Generate a key pair for the client
	clientPrivKey, clientPubKey, err := utils.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate a client ID
	clientID := utils.PublicKeyToAddress(clientPubKey).Hex()

	// Create the session
	now := time.Now()
	session := &Session{
		ID:            id,
		PairingTopic:  pairingTopic,
		SessionTopic:  sessionTopic,
		SymKey:        symKey,
		ClientID:      clientID,
		ClientPubKey:  clientPubKey,
		ClientPrivKey: clientPrivKey,
		Status:        SessionStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
		ExpiresAt:     now.Add(24 * time.Hour), // Sessions expire after 24 hours
	}

	return session, nil
}

// GeneratePairingURI generates a pairing URI for the session
// By default, this URI does NOT include the relay server URL, and the wallet app will use its own default relay server.
// Format: wc:{topic}@2?relay-protocol=irn&symKey={key}
// If includeRelayURL is true, it will add the relay-url parameter.
func (s *Session) GeneratePairingURI() string {
	// WalletConnect v2 format - does not include relay URL, only the protocol
	uri := fmt.Sprintf("wc:%s@2?relay-protocol=irn&symKey=%s", s.PairingTopic, s.SymKey)
	return uri
}

// GeneratePairingURIWithRelay generates a pairing URI that includes the relay URL
func (s *Session) GeneratePairingURIWithRelay(relayURL string) string {
	// URL encode the relay URL
	encodedRelayURL := url.QueryEscape(relayURL)
	// WalletConnect v2 format with custom relay URL
	uri := fmt.Sprintf("wc:%s@2?relay-protocol=irn&relay-url=%s&symKey=%s",
		s.PairingTopic, encodedRelayURL, s.SymKey)
	return uri
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// SetWalletAddress sets the wallet address for the session
func (s *Session) SetWalletAddress(address common.Address) {
	s.WalletAddress = address
	s.UpdatedAt = time.Now()
}

// SetPeerID sets the peer ID for the session
func (s *Session) SetPeerID(peerID string) {
	s.PeerID = peerID
	s.UpdatedAt = time.Now()
}

// SetPeerPubKey sets the peer public key for the session
func (s *Session) SetPeerPubKey(pubKey *ecdsa.PublicKey) {
	s.PeerPubKey = pubKey
	s.UpdatedAt = time.Now()
}

// SetStatus sets the status of the session
func (s *Session) SetStatus(status SessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()
}

// Activate activates the session
func (s *Session) Activate() {
	s.Status = SessionStatusActive
	s.UpdatedAt = time.Now()
}

// Disconnect disconnects the session
func (s *Session) Disconnect() {
	s.Status = SessionStatusDisconnected
	s.UpdatedAt = time.Now()
}

// ToJSON converts the session to JSON
func (s *Session) ToJSON() (string, error) {
	// Create a copy of the session without the private key
	sessionCopy := *s
	sessionCopy.ClientPrivKey = nil

	// Convert the public keys to hex strings
	type sessionJSON struct {
		ID            string        `json:"id"`
		PairingTopic  string        `json:"pairing_topic"`
		SessionTopic  string        `json:"session_topic"`
		SymKey        string        `json:"sym_key"`
		ClientID      string        `json:"client_id"`
		PeerID        string        `json:"peer_id"`
		ClientPubKey  string        `json:"client_pub_key"`
		PeerPubKey    string        `json:"peer_pub_key,omitempty"`
		WalletAddress string        `json:"wallet_address"`
		Status        SessionStatus `json:"status"`
		CreatedAt     time.Time     `json:"created_at"`
		UpdatedAt     time.Time     `json:"updated_at"`
		ExpiresAt     time.Time     `json:"expires_at"`
	}

	jsonSession := sessionJSON{
		ID:            sessionCopy.ID,
		PairingTopic:  sessionCopy.PairingTopic,
		SessionTopic:  sessionCopy.SessionTopic,
		SymKey:        sessionCopy.SymKey,
		ClientID:      sessionCopy.ClientID,
		PeerID:        sessionCopy.PeerID,
		ClientPubKey:  utils.PublicKeyToHex(sessionCopy.ClientPubKey),
		WalletAddress: sessionCopy.WalletAddress.Hex(),
		Status:        sessionCopy.Status,
		CreatedAt:     sessionCopy.CreatedAt,
		UpdatedAt:     sessionCopy.UpdatedAt,
		ExpiresAt:     sessionCopy.ExpiresAt,
	}

	if sessionCopy.PeerPubKey != nil {
		jsonSession.PeerPubKey = utils.PublicKeyToHex(sessionCopy.PeerPubKey)
	}

	bytes, err := json.Marshal(jsonSession)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	return string(bytes), nil
}

// SessionManager manages WalletConnect sessions
type SessionManager struct {
	sessions map[string]*Session // session ID -> session
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session
func (m *SessionManager) CreateSession() (*Session, error) {
	session, err := NewSession()
	if err != nil {
		return nil, err
	}

	m.sessions[session.ID] = session
	return session, nil
}

// GetSession gets a session by ID
func (m *SessionManager) GetSession(id string) *Session {
	return m.sessions[id]
}

// GetSessionByPairingTopic gets a session by pairing topic
func (m *SessionManager) GetSessionByPairingTopic(topic string) *Session {
	for _, session := range m.sessions {
		if session.PairingTopic == topic {
			return session
		}
	}
	return nil
}

// GetSessionBySessionTopic gets a session by session topic
func (m *SessionManager) GetSessionBySessionTopic(topic string) *Session {
	for _, session := range m.sessions {
		if session.SessionTopic == topic {
			return session
		}
	}
	return nil
}

// RemoveSession removes a session
func (m *SessionManager) RemoveSession(id string) {
	delete(m.sessions, id)
}

// GetActiveSessions gets all active sessions
func (m *SessionManager) GetActiveSessions() []*Session {
	var activeSessions []*Session
	for _, session := range m.sessions {
		if session.Status == SessionStatusActive && !session.IsExpired() {
			activeSessions = append(activeSessions, session)
		}
	}
	return activeSessions
}

// CleanupExpiredSessions removes expired sessions
func (m *SessionManager) CleanupExpiredSessions() {
	for id, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, id)
		}
	}
}
