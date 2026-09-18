package utils

import "testing"

func TestMaskInputKeys(t *testing.T) {
	payload := `{"id":"1","input":{"api_key":"secret","nested":{"api_key":"kept"},"name":"secret"},"output":{"api_key":"kept"}}`
	got := MaskInputKeys([]byte(payload), []string{"api_key", "missing"})
	want := `{"id":"1","input":{"api_key":"***","name":"secret","nested":{"api_key":"kept"}},"output":{"api_key":"kept"}}`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestMaskValues(t *testing.T) {
	input := map[string]interface{}{"api_key": "secret", "token": "secret-longer", "count": 3, "empty": ""}
	values := SensitiveValues(input, []string{"api_key", "token", "count", "empty", "missing"})
	got := MaskValues("key=secret token=secret-longer count=3", values)
	want := "key=*** token=*** count=3"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
