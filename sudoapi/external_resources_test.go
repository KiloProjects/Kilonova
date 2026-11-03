package sudoapi

import (
	"testing"

	"github.com/KiloProjects/kilonova"
)

func TestExternalResourceValidation(t *testing.T) {
	for _, rawURL := range []string{"javascript:alert(1)", "data:text/html,bad", "//example.com", "/relative", "https://user:password@example.com", "ftp://example.com", "https://"} {
		if err := validateExternalResource("Editorial", rawURL, "en", kilonova.ResourceTypeEditorial); err == nil {
			t.Errorf("unsafe URL accepted: %q", rawURL)
		}
	}
	for _, language := range []string{"", "en", "ro"} {
		if err := validateExternalResource("Editorial", "https://example.com/editorial?lang=en#solution", language, kilonova.ResourceTypeEditorial); err != nil {
			t.Errorf("valid resource rejected: %v", err)
		}
	}
	if err := validateExternalResource("Editorial", "https://example.com", "invalid", kilonova.ResourceTypeVideo); err == nil {
		t.Error("invalid language accepted")
	}
	if err := validateExternalResource("Editorial", "https://example.com", "en", "invalid"); err == nil {
		t.Error("invalid type accepted")
	}
	if err := validateExternalResource("  ", "https://example.com", "en", kilonova.ResourceTypeVideo); err == nil {
		t.Error("empty name accepted")
	}
}
