package contestgen

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintablePDF(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed")
	}
	for name, csv := range map[string]string{
		"teams":    "Team Name,Members,Contact Email,Username,Password\nFoo,\"a, b\",x@y.z,foo1,pw1\n",
		"no teams": "Username,Password\nfoo1,pw1\n",
	} {
		t.Run(name, func(t *testing.T) {
			pdf, err := PrintablePDF(t.Context(), []byte(csv))
			require.NoError(t, err)
			require.True(t, bytes.HasPrefix(pdf, []byte("%PDF")))
		})
	}
}
