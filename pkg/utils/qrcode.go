package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRCode generates a QR code for the given content
func GenerateQRCode(content string, size int) (string, error) {
	if size <= 0 {
		size = 256 // Default size
	}

	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Create a buffer to store the PNG image
	var buf bytes.Buffer
	err = qr.Write(size, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to write QR code: %w", err)
	}

	// Encode the image as base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}
