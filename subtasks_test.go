package kilonova

import (
	"log"
	"testing"
)

func TestParseSubtaskString(t *testing.T) {
	var candidates []string = []string{
		"1-3;f4-5;t6-7",
		"3-5f;1-2;3-4",
	}

	for _, c := range candidates {
		log.Println(c)
		ParseSubtaskString(c)
	}
}
