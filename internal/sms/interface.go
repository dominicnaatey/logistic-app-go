package sms

// Sender defines the SMS operations the application depends on.
// Service (Africa's Talking) implements this interface.
// Swap to Twilio, AWS SNS, or a mock by implementing these three methods.
type Sender interface {
	SendOTP(phone, code string) error
	SendWelcome(phone, name, lang string) error
	SendTripAssignment(phone, driverName, origin, destination string) error
}
