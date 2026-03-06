package kwtsms

import (
	"os"
	"strings"
)

// loadEnvFile parses a .env file into a map of key=value pairs.
// Returns an empty map if the file does not exist or cannot be read.
// Never panics or returns an error.
//
// Parsing rules:
//   - Ignores blank lines and lines starting with #
//   - Strips inline # comments from unquoted values
//   - Supports quoted values: KWTSMS_SENDER_ID="MY APP" -> MY APP
//   - Does NOT modify os environment variables (read-only parsing)
func loadEnvFile(path string) map[string]string {
	env := make(map[string]string)

	data, err := os.ReadFile(path)
	if err != nil {
		return env
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip one matching outer quote pair
		if len(val) >= 2 && val[0] == val[len(val)-1] && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		} else {
			// Strip inline # comments for unquoted values
			if ci := strings.Index(val, " #"); ci >= 0 {
				val = strings.TrimSpace(val[:ci])
			}
		}

		env[key] = val
	}

	return env
}
