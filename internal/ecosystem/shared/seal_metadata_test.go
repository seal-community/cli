package shared

import (
	"fmt"
	"strings"
	"testing"
)


func TestEmptyMetadataFile(t *testing.T) {
	content := ``
	_, err := LoadPackageMetadata(strings.NewReader(content))
	if err == nil {
		t.Fatalf("should not parse empty metadata %v", err)
	}
}

func TestNoExtraFields(t *testing.T) {
	content := `aaa:123`
	metadata, err := LoadPackageMetadata(strings.NewReader(content))

	if metadata != nil {
		t.Fatalf("allowed extraneous field in metadata: %v", metadata)
	}

	if err == nil {
		t.Fatalf("should fail parsing yaml with extraneous field")
	}
}

func TestNoDupFields(t *testing.T) {
	firstVal := "1.2.3"
	secondVal := "4.5.6"
	content := fmt.Sprintf("version: %s\nversion: %s", firstVal, secondVal)
	metadata, err := LoadPackageMetadata(strings.NewReader(content))

	if metadata != nil {
		t.Fatalf("allowed duplicate field in metadata: %v", metadata)
	}

	if err == nil {
		t.Fatalf("should fail parsing yaml with dup fields field: %v", err)
	}
}

func TestSanity(t *testing.T) {
	versionValue := "1.2.3"
	content := fmt.Sprintf("version: %s", versionValue)
	metadata, err := LoadPackageMetadata(strings.NewReader(content))

	if metadata == nil {
		t.Fatalf("failed loading metadata: %v", err)
	}

	if metadata.SealedVersion != versionValue {
		t.Fatalf("failed parsing content - got %s expected %s", metadata.SealedVersion, versionValue)
	}
}
