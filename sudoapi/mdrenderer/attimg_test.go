package mdrenderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			rendered := string(got)
			assert.Contains(t, rendered, tc.wantAlign)
			if tc.unexpected != "" {
				assert.NotContains(t, rendered, tc.unexpected)
			}
		})
	}
}
