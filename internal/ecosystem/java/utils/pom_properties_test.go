package utils

import (
	"io"
	"strings"
	"testing"
)

func TestReadPomPropertiesFromFile(t *testing.T) {
	reader := strings.NewReader(`groupId=com.example
artifactId=example
version=1.0.0
`)
	pomProperties := ReadPomPropertiesFromFile(reader)
	if pomProperties == nil {
		t.Fatalf("failed to read pom properties")
	}

	if pomProperties.GroupId != "com.example" || pomProperties.ArtifactId != "example" || pomProperties.Version != "1.0.0" {
		t.Fatalf("failed to read pom properties")
	}
}

func TestReadPomPropertiesFromFileMissingValue(t *testing.T) {
	reader := strings.NewReader(`groupId=com.example
version=1.0.0
`)
	pomProperties := ReadPomPropertiesFromFile(reader)
	if pomProperties != nil {
		t.Fatalf("should have failed to read pom properties")
	}
}

func TestPomPropertiesGetAsReader(t *testing.T) {
	pomProperties := &PomProperties{
		GroupId:    "com.example",
		ArtifactId: "example",
		Version:    "1.0.0",
	}
	reader := pomProperties.GetAsReader()
	data, _ := io.ReadAll(reader)
	expected := `artifactId=example
groupId=com.example
version=1.0.0
`
	if string(data) != expected {
		t.Fatalf("failed to get reader for pom properties")
	}
}

func TestGetPackageId(t *testing.T) {
	pomProperties := &PomProperties{
		GroupId:    "com.example",
		ArtifactId: "example",
		Version:    "1.0.0",
	}
	packageId := pomProperties.GetPackageId()
	if packageId != "Maven|com.example:example@1.0.0" {
		t.Fatalf("failed to get package id")
	}
}
