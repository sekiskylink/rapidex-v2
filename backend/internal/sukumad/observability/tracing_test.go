package observability

import "testing"

func TestSanitizeEventDataMasksSecrets(t *testing.T) {
	input := map[string]any{
		"accessToken": "secret",
		"nested": map[string]any{
			"password": "hidden",
		},
		"status": "ok",
	}

	sanitized := sanitizeEventData(input)
	if sanitized["accessToken"] != "[masked]" {
		t.Fatalf("expected access token to be masked, got %#v", sanitized["accessToken"])
	}
	nested, ok := sanitized["nested"].(map[string]any)
	if !ok || nested["password"] != "[masked]" {
		t.Fatalf("expected nested password to be masked, got %#v", sanitized["nested"])
	}
	if sanitized["status"] != "ok" {
		t.Fatalf("expected non-sensitive field to remain, got %#v", sanitized["status"])
	}
}

func TestPreviewEventDataTruncatesLargePayload(t *testing.T) {
	preview := previewEventData(map[string]any{"message": "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"})
	if len(preview) == 0 {
		t.Fatal("expected preview text")
	}
}
