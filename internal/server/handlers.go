package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/korjavin/wctestapp/pkg/utils"
)

// TemplateData represents the data passed to templates
type TemplateData struct {
	Title            string
	QRCode           string
	PairingURI       string
	SessionID        string
	WalletAddress    string
	Message          string
	Signature        string
	SignatureDetails map[string]string
	Error            string
}

// handleIndex handles the index page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Parse the template
	tmpl, err := template.ParseFiles(
		filepath.Join(s.config.TemplateDir, "layout.html"),
		filepath.Join(s.config.TemplateDir, "index.html"),
	)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to parse template: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the template
	err = tmpl.ExecuteTemplate(w, "layout", TemplateData{
		Title: "WalletConnect Test App",
	})
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to render template: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleConnected handles the connected page
func (s *Server) handleConnected(w http.ResponseWriter, r *http.Request) {
	// Get the session ID from the query parameters
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get the session
	session := s.walletClient.GetSession(sessionID)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get the message and signature from the query parameters
	message := r.URL.Query().Get("message")
	signature := r.URL.Query().Get("signature")

	// Get signature details if available
	var signatureDetails map[string]string
	if message != "" && signature != "" {
		var err error
		signatureDetails, err = s.walletClient.GetSignatureDetails(message, signature)
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to get signature details: %v", err))
		}
	}

	// Parse the template
	tmpl, err := template.ParseFiles(
		filepath.Join(s.config.TemplateDir, "layout.html"),
		filepath.Join(s.config.TemplateDir, "connected.html"),
	)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to parse template: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the template
	err = tmpl.ExecuteTemplate(w, "layout", TemplateData{
		Title:            "Connected Wallet",
		SessionID:        sessionID,
		WalletAddress:    session.WalletAddress.Hex(),
		Message:          message,
		Signature:        signature,
		SignatureDetails: signatureDetails,
	})
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to render template: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleCreateSession handles the create session API endpoint
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a new session
	session, err := s.walletClient.CreateSession()
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to create session: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Generate the pairing URI
	pairingURI := session.GeneratePairingURI()

	// Log the pairing URI for debugging
	s.logger.Info(fmt.Sprintf("Pairing URI: %s", pairingURI))

	// Generate a QR code for the pairing URI
	qrCode, err := utils.GenerateQRCode(pairingURI, 256)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to generate QR code: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Connect to the relay server
	err = s.walletClient.ConnectToRelay(session)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to connect to relay: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Return the session details
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id":  session.ID,
		"pairing_uri": pairingURI,
		"qr_code":     qrCode,
	})
}

// handleSessionStatus handles the session status API endpoint
func (s *Server) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	// Get the session ID from the query parameters
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Get the session
	session := s.walletClient.GetSession(sessionID)
	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Return the session status
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id":     session.ID,
		"status":         session.Status,
		"wallet_address": session.WalletAddress.Hex(),
	})
}

// handleDisconnectSession handles the disconnect session API endpoint
func (s *Server) handleDisconnectSession(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the session ID from the query parameters
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Get the session
	session := s.walletClient.GetSession(sessionID)
	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Disconnect the session
	err := s.walletClient.DisconnectSession(session)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to disconnect session: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Return success
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleSignMessage handles the sign message API endpoint
func (s *Server) handleSignMessage(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var request struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if request.SessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}
	if request.Message == "" {
		http.Error(w, "Missing message", http.StatusBadRequest)
		return
	}

	// Get the session
	session := s.walletClient.GetSession(request.SessionID)
	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Check if the session is active
	if session.Status != "active" {
		http.Error(w, "Session is not active", http.StatusBadRequest)
		return
	}

	// Sign the message
	signature, err := s.walletClient.SignMessage(session, request.Message)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to sign message: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "application/json")

	// Return the signature
	json.NewEncoder(w).Encode(map[string]interface{}{
		"signature": signature,
	})
}

// GetSignatureDetails gets the details of a signature
func (s *Server) GetSignatureDetails(message, signature string) (map[string]string, error) {
	return s.walletClient.GetSignatureDetails(message, signature)
}
