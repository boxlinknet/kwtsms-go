package kwtsms

import "testing"

func TestEnrichErrorAddsAction(t *testing.T) {
	data := map[string]any{
		"result":      "ERROR",
		"code":        "ERR003",
		"description": "Authentication error",
	}
	enriched := EnrichError(data)
	action, ok := enriched["action"].(string)
	if !ok || action == "" {
		t.Fatal("expected action to be added")
	}
	if action != APIErrors["ERR003"] {
		t.Errorf("action = %q, want %q", action, APIErrors["ERR003"])
	}
}

func TestEnrichErrorNoEffectOnOK(t *testing.T) {
	data := map[string]any{
		"result": "OK",
		"code":   "ERR003",
	}
	enriched := EnrichError(data)
	if _, ok := enriched["action"]; ok {
		t.Error("EnrichError should not add action to OK responses")
	}
}

func TestEnrichErrorUnknownCode(t *testing.T) {
	data := map[string]any{
		"result":      "ERROR",
		"code":        "ERR999",
		"description": "Unknown error",
	}
	enriched := EnrichError(data)
	if _, ok := enriched["action"]; ok {
		t.Error("EnrichError should not add action for unknown error codes")
	}
}

func TestEnrichErrorDoesNotMutateOriginal(t *testing.T) {
	data := map[string]any{
		"result": "ERROR",
		"code":   "ERR003",
	}
	EnrichError(data)
	if _, ok := data["action"]; ok {
		t.Error("EnrichError should not mutate the original map")
	}
}

func TestAPIErrorsHasAllCodes(t *testing.T) {
	expected := []string{
		"ERR001", "ERR002", "ERR003", "ERR004", "ERR005",
		"ERR006", "ERR007", "ERR008", "ERR009", "ERR010",
		"ERR011", "ERR012", "ERR013", "ERR019", "ERR020",
		"ERR021", "ERR022", "ERR023", "ERR024", "ERR025",
		"ERR026", "ERR027", "ERR028", "ERR029", "ERR030",
		"ERR031", "ERR032", "ERR033", "ERR_INVALID_INPUT",
	}
	for _, code := range expected {
		if _, ok := APIErrors[code]; !ok {
			t.Errorf("APIErrors missing code %q", code)
		}
	}
}
