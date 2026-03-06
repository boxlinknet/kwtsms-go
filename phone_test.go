package kwtsms

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"+96598765432", "96598765432"},
		{"0096598765432", "96598765432"},
		{"965 9876 5432", "96598765432"},
		{"965-9876-5432", "96598765432"},
		{"(965) 9876-5432", "96598765432"},
		{"٩٦٥٩٨٧٦٥٤٣٢", "96598765432"},     // Arabic-Indic digits
		{"۹۶۵۹۸۷۶۵۴۳۲", "96598765432"},     // Extended Arabic-Indic / Persian
		{"+965.9876.5432", "96598765432"},
		{"", ""},
		{"   ", ""},
		{"00096598765432", "96598765432"}, // triple leading zeros
		{"96598765432", "96598765432"},    // already clean
		{"  +965 9876 5432  ", "96598765432"},
		{"abc", ""},
		{"---", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePhone(tt.input)
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePhoneInput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantValid  bool
		wantNorm   string
		wantErrSub string // substring expected in error message
	}{
		{"valid Kuwait", "+96598765432", true, "96598765432", ""},
		{"valid 00 prefix", "0096598765432", true, "96598765432", ""},
		{"valid Arabic digits", "٩٦٥٩٨٧٦٥٤٣٢", true, "96598765432", ""},
		{"valid minimum 7 digits", "1234567", true, "1234567", ""},
		{"valid maximum 15 digits", "123456789012345", true, "123456789012345", ""},
		{"empty", "", false, "", "required"},
		{"blank spaces", "   ", false, "", "required"},
		{"email address", "user@gmail.com", false, "", "email address"},
		{"no digits", "abc", false, "", "no digits found"},
		{"dashes only", "---", false, "", "no digits found"},
		{"too short 3 digits", "123", false, "123", "too short"},
		{"too short 6 digits", "123456", false, "123456", "too short"},
		{"too long 16 digits", "1234567890123456", false, "1234567890123456", "too long"},
		{"too short 1 digit", "5", false, "5", "1 digit"},
		{"email with numbers", "test123@gmail.com", false, "", "email address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ValidatePhoneInput(tt.input)
			if v.Valid != tt.wantValid {
				t.Errorf("ValidatePhoneInput(%q).Valid = %v, want %v", tt.input, v.Valid, tt.wantValid)
			}
			if tt.wantNorm != "" && v.Normalized != tt.wantNorm {
				t.Errorf("ValidatePhoneInput(%q).Normalized = %q, want %q", tt.input, v.Normalized, tt.wantNorm)
			}
			if tt.wantErrSub != "" && !contains(v.Error, tt.wantErrSub) {
				t.Errorf("ValidatePhoneInput(%q).Error = %q, want substring %q", tt.input, v.Error, tt.wantErrSub)
			}
			if tt.wantValid && v.Error != "" {
				t.Errorf("ValidatePhoneInput(%q).Error = %q, want empty for valid input", tt.input, v.Error)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || containsSub(s, substr))
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
