package kwtsms

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	content := `# Comment line
KWTSMS_USERNAME=myuser
KWTSMS_PASSWORD=mypass
KWTSMS_SENDER_ID="MY APP"
KWTSMS_TEST_MODE=1
KWTSMS_LOG_FILE=custom.log

# Another comment
EMPTY_VAL=
QUOTED_SINGLE='hello world'
WITH_COMMENT=value  # this is a comment
SPACES_AROUND = spaced_value
`
	os.WriteFile(path, []byte(content), 0644)

	env := loadEnvFile(path)

	tests := []struct {
		key  string
		want string
	}{
		{"KWTSMS_USERNAME", "myuser"},
		{"KWTSMS_PASSWORD", "mypass"},
		{"KWTSMS_SENDER_ID", "MY APP"},
		{"KWTSMS_TEST_MODE", "1"},
		{"KWTSMS_LOG_FILE", "custom.log"},
		{"EMPTY_VAL", ""},
		{"QUOTED_SINGLE", "hello world"},
		{"WITH_COMMENT", "value"},
		{"SPACES_AROUND", "spaced_value"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := env[tt.key]
			if !ok {
				t.Fatalf("key %q not found in env", tt.key)
			}
			if got != tt.want {
				t.Errorf("env[%q] = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestLoadEnvFileMissing(t *testing.T) {
	env := loadEnvFile("/nonexistent/path/.env")
	if len(env) != 0 {
		t.Errorf("expected empty map for missing file, got %v", env)
	}
}

func TestLoadEnvFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte(""), 0644)

	env := loadEnvFile(path)
	if len(env) != 0 {
		t.Errorf("expected empty map for empty file, got %v", env)
	}
}
