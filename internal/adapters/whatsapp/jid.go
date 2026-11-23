package whatsapp

import (
	"errors"
	"strings"
	"unicode"

	"go.mau.fi/whatsmeow/types"
)

var (
	ErrInvalidPhoneNumber = errors.New("whatsapp: phone number must contain digits")
	ErrEmptyMessageBody   = errors.New("whatsapp: message body must not be empty")
	ErrMissingSessionDB   = errors.New("whatsapp: session DB path is required")
)

// PhoneToJID normalizes an E.164 or formatted phone number into a standard 1:1 WhatsApp JID.
func PhoneToJID(phone string) (types.JID, error) {
	var digits strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	if digits.Len() == 0 {
		return types.EmptyJID, ErrInvalidPhoneNumber
	}
	return types.NewJID(digits.String(), types.DefaultUserServer), nil
}

// JIDToPhone converts a WhatsApp user JID back into a normalized E.164 phone number ("+<digits>").
func JIDToPhone(jid types.JID) string {
	user := strings.TrimSpace(jid.User)
	if user == "" {
		return ""
	}
	if strings.HasPrefix(user, "+") {
		return user
	}
	return "+" + user
}
