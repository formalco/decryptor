package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCORSAllowedOrigin(t *testing.T) {
	valid := map[string]string{
		"":                              defaultCORSAllowedOrigin,
		"https://app.formal.ai":         "https://app.formal.ai",
		"https://console.example.com":   "https://console.example.com",
		"https://console.internal:8443": "https://console.internal:8443",
	}
	for raw, want := range valid {
		got, err := parseCORSAllowedOrigin(raw)
		require.NoError(t, err, raw)
		require.Equal(t, want, got, raw)
	}

	invalid := []string{
		"*",
		"app.formal.ai",
		"http://app.formal.ai",
		"https://",
		"https://app.formal.ai/",
		"https://app.formal.ai/logs",
		"https://app.formal.ai?x=1",
		"https://app.formal.ai#x",
		"https://user@app.formal.ai",
		"https://App.Formal.ai",
		"https://app.formal.ai:443",
		"https://a.example.com,https://b.example.com",
		" https://app.formal.ai",
	}
	for _, raw := range invalid {
		_, err := parseCORSAllowedOrigin(raw)
		require.Error(t, err, raw)
	}
}
