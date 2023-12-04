package pnpm

import (
	"testing"
)

func TestPnpmOutputSkipping(t *testing.T) {
	before := ""
	after := skipUntilJsonStarts(before)
	if after != "" {
		t.Fatalf("skip failed - returning empty")
	}
}

func TestPnpmOutputSkippingValid(t *testing.T) {
	before := "[]"
	after := skipUntilJsonStarts(before)
	if after != before {
		t.Fatalf("skip failed - should not change input, before: `%s` after: `%s`", before, after)
	}
}

func TestPnpmOutputSkippingUnsupported(t *testing.T) {
	before := "abcdef"
	after := skipUntilJsonStarts(before)
	if after != "" {
		t.Fatalf("should not skip, before: `%s` after: `%s`", before, after)
	}
}

func TestPnpmOutputSkippingInvalid(t *testing.T) {
	before := " WARN  Issue while reading \"/Users/mococo/proj/.npmrc\". Failed to replace env in config: ${VARNAME}\n[]"
	after := skipUntilJsonStarts(before)
	if after != "[]" {
		t.Fatalf("skip failed - did not skip correctly, before: `%s` after: `%s`", before, after)
	}
}

func TestPnpmOutputSkippingInvalidCR(t *testing.T) {
	// should work regardless of os since \n is after \r
	before := " WARN  Issue while reading \"/Users/mococo/proj/.npmrc\". Failed to replace env in config: ${VARNAME}\r\n[]"
	after := skipUntilJsonStarts(before)
	if after != "[]" {
		t.Fatalf("skip failed - did not skip correctly, before: `%s` after: `%s`", before, after)
	}
}
