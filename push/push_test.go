package push

import (
	"testing"
)

func TestCustomPushServerURL(t *testing.T) {

	t.Run("with path fails", func(t *testing.T) {
		err := ValidateCustomPushServerURL("https://example.com/path/to/push")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("without path succeeds", func(t *testing.T) {
		err := ValidateCustomPushServerURL("https://example.com:1234")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("without schema fails", func(t *testing.T) {
		err := ValidateCustomPushServerURL("example.com:1234")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
