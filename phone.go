package kwtsms

import (
	"fmt"
	"strings"
	"unicode"
)

// arabicToLatin maps Arabic-Indic (U+0660-U+0669) and Extended Arabic-Indic /
// Persian (U+06F0-U+06F9) digits to their Latin equivalents.
var arabicToLatin = map[rune]rune{
	'٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4',
	'٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9',
	'۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4',
	'۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9',
}

// NormalizePhone converts a raw phone number to kwtSMS-accepted format:
// digits only, no leading zeros.
//
// Steps: convert Arabic/Persian digits to Latin, strip all non-digit
// characters, strip leading zeros.
func NormalizePhone(phone string) string {
	var b strings.Builder
	b.Grow(len(phone))
	for _, r := range phone {
		if latin, ok := arabicToLatin[r]; ok {
			b.WriteRune(latin)
		} else if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimLeft(b.String(), "0")
}

// PhoneValidation holds the result of ValidatePhoneInput.
type PhoneValidation struct {
	Valid      bool
	Error      string
	Normalized string
}

// ValidatePhoneInput validates a raw phone number input before sending to the API.
//
// Catches every common mistake without panicking:
//   - Empty or blank input
//   - Email address entered instead of a phone number
//   - Non-numeric text with no digits
//   - Too short after normalization (< 7 digits)
//   - Too long after normalization (> 15 digits, E.164 maximum)
func ValidatePhoneInput(phone string) PhoneValidation {
	raw := strings.TrimSpace(fmt.Sprintf("%v", phone))

	if raw == "" {
		return PhoneValidation{Valid: false, Error: "Phone number is required", Normalized: ""}
	}

	if strings.Contains(raw, "@") {
		return PhoneValidation{
			Valid: false,
			Error: fmt.Sprintf("'%s' is an email address, not a phone number", raw),
		}
	}

	normalized := NormalizePhone(raw)

	if normalized == "" {
		return PhoneValidation{
			Valid: false,
			Error: fmt.Sprintf("'%s' is not a valid phone number, no digits found", raw),
		}
	}

	n := len(normalized)
	if n < 7 {
		digit := "digits"
		if n == 1 {
			digit = "digit"
		}
		return PhoneValidation{
			Valid:      false,
			Error:      fmt.Sprintf("'%s' is too short to be a valid phone number (%d %s, minimum is 7)", raw, n, digit),
			Normalized: normalized,
		}
	}

	if n > 15 {
		return PhoneValidation{
			Valid:      false,
			Error:      fmt.Sprintf("'%s' is too long to be a valid phone number (%d digits, maximum is 15)", raw, n),
			Normalized: normalized,
		}
	}

	return PhoneValidation{Valid: true, Normalized: normalized}
}
