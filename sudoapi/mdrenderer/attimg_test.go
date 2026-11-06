package mdrenderer

import (
	"strings"
	"testing"
)

func TestAttachmentAlignValidation(t *testing.T) {
	tests := []struct {
		name       string
		markdown   string
		wantAlign  string
		unexpected string
	}{
		{
			name:      "supported alignment",
			markdown:  `~[image.png|align=right]`,
			wantAlign: `data-imgalign="right"`,
		},
		{
			name:       "attribute injection",
			markdown:   `~[missing.png|align=x" onerror="alert(document.cookie)]`,
			wantAlign:  `data-imgalign="left"`,
			unexpected: `onerror=`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewRenderer().Render([]byte(tc.markdown), nil)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			rendered := string(got)
			if !strings.Contains(rendered, tc.wantAlign) {
				t.Errorf("Render() = %q, want %q", rendered, tc.wantAlign)
			}
			if tc.unexpected != "" && strings.Contains(rendered, tc.unexpected) {
				t.Errorf("Render() = %q, unexpectedly contains %q", rendered, tc.unexpected)
			}
		})
	}
}
