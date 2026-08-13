package sms

import (
	"fmt"
	"log"
	"strings"

	atSMS "github.com/AfricasTalkingLtd/africastalking-go/sms"
)

const (
	// envSandbox is the Africa's Talking sandbox environment identifier.
	envSandbox = "sandbox"

	// envProduction is the Africa's Talking production environment identifier.
	envProduction = "production"
)

// Service wraps the Africa's Talking SMS SDK.
// In sandbox mode messages are NOT sent to real phones — they appear
// in the AT simulator at https://simulator.africastalking.com
type Service struct {
	client    atSMS.Service
	senderID  string // optional shortcode / alphanumeric sender ID
	isSandbox bool
}

// NewService creates and returns a configured SMS service.
// username and apiKey come from the Africa's Talking dashboard.
// Set username = "sandbox" and use the sandbox API key for development.
func NewService(username, apiKey, senderID string) (*Service, error) {
	if username == "" || apiKey == "" {
		return nil, fmt.Errorf("Africa's Talking username and API key are required")
	}

	// AT SDK uses "sandbox" string to switch environments
	env := envProduction
	isSandbox := false
	if strings.ToLower(username) == envSandbox {
		env = envSandbox
		isSandbox = true
	}

	client := atSMS.NewService(username, apiKey, env)

	svc := &Service{
		client:    client,
		senderID:  senderID,
		isSandbox: isSandbox,
	}

	if isSandbox {
		log.Println("✓ SMS service initialised (SANDBOX — messages go to AT simulator)")
	} else {
		log.Println("✓ SMS service initialised (PRODUCTION)")
	}

	return svc, nil
}

// SendOTP delivers a 6-digit OTP to the given phone number.
// Phone must be in E.164 format (e.g. +233501234567).
// The message is intentionally short to keep SMS costs low and
// to fit within a single SMS unit (160 chars).
func (s *Service) SendOTP(phone, code string) error {
	message := fmt.Sprintf("Your verification code is %s. Valid for 10 minutes. Do not share this code.", code)
	return s.send(phone, message)
}

// SendWelcome sends a welcome message after a user's first login.
// lang should be "en" or "fr" — the message is localised accordingly.
func (s *Service) SendWelcome(phone, name, lang string) error {
	var message string
	switch lang {
	case "fr":
		message = fmt.Sprintf("Bienvenue sur LogiTrack, %s! Votre compte a été créé avec succès.", name)
	default: // "en"
		message = fmt.Sprintf("Welcome to LogiTrack, %s! Your account has been created successfully.", name)
	}
	return s.send(phone, message)
}

// SendTripAssignment notifies a driver that a new trip has been assigned.
func (s *Service) SendTripAssignment(phone, driverName, origin, destination string) error {
	message := fmt.Sprintf(
		"Hi %s, a new trip has been assigned: %s → %s. Open the app to accept or reject.",
		driverName, origin, destination,
	)
	return s.send(phone, message)
}

// IsSandbox returns true when the service is running in sandbox mode.
func (s *Service) IsSandbox() bool {
	return s.isSandbox
}

// send is the internal method that calls the AT SDK.
// It handles the senderID (empty string = AT default sender).
func (s *Service) send(phone, message string) error {
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}
	if message == "" {
		return fmt.Errorf("message body is required")
	}

	resp, err := s.client.Send(s.senderID, phone, message)
	if err != nil {
		return fmt.Errorf("AT SDK error: %w", err)
	}

	// Check AT-level delivery status per recipient
	if resp != nil && resp.SMS.Recipients != nil {
		for _, r := range resp.SMS.Recipients {
			if r.Status != "Success" {
				return fmt.Errorf("SMS delivery failed for %s: %s (cost: %s)",
					r.Number, r.Status, r.Cost)
			}
		}
	}

	if s.isSandbox {
		log.Printf("📱 [SANDBOX] SMS sent to %s — check AT simulator", phone)
	} else {
		log.Printf("📱 SMS sent to %s", phone)
	}

	return nil
}
