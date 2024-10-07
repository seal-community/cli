package composer

import (
	"testing"
)

type TestPair struct {
	input    string
	expected string
}

func TestGetMetadataDepFileSanityWindows(t *testing.T) {
	output := getMetadataDepFile("C:\\a\\b\\c", "vendor/package")
	expected := "C:\\a\\b\\c\\vendor\\vendor\\package\\.seal-metadata.yaml"
	if output != expected {
		t.Fatalf("got %s, expected %s", output, expected)
	}
}
