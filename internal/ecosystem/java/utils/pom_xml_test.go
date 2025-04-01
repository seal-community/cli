package utils

import (
	"io"
	"strings"
	"testing"
)

const pomXMLParentGroupID = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <parent>
    <groupId>com.fasterxml.jackson</groupId>
    <artifactId>jackson-base</artifactId>
    <version>2.13.1</version>
  </parent>
  <artifactId>jackson-databind</artifactId>
  <version>2.13.1</version>
</project>`

const pomXMLNoParent = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <groupId>com.fasterxml.jackson</groupId>
  <artifactId>jackson-databind</artifactId>
  <version>2.13.1</version>
</project>`

const pomXMLParentVersion = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <parent>
    <groupId>com.fasterxml.jackson</groupId>
    <artifactId>jackson-base</artifactId>
    <version>2.13.1</version>
  </parent>
  <groupId>com.fasterxml.jackson</groupId>
  <artifactId>jackson-databind</artifactId>
</project>`

// based on https://repo1.maven.org/maven2/org/springframework/retry/spring-retry/2.0.10/spring-retry-2.0.10.pom
const pomXMLWithProperties = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${revision}</version>
<name>Spring Retry</name>
<properties>
<revision>2.13.1</revision>
</properties>
</project>`

const pomXMLWithMultipleProperties = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${major}.${minor}.${patch}</version>
<name>Spring Retry</name>
<properties>
<major>2</major>
<minor>13</minor>
<patch>1</patch>
</properties>
</project>`

const pomXMLWithMultiplePropertiesRemainder = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${major}.${minor}.1</version>
<name>Spring Retry</name>
<properties>
<major>2</major>
<minor>13</minor>
</properties>
</project>`

// this also tests the GetPackageId method since it's the only way to test it
func TestReadPomXMLFromFile(t *testing.T) {
	tests := []struct {
		xml string
	}{
		{pomXMLParentGroupID},
		{pomXMLNoParent},
		{pomXMLParentVersion},
		{pomXMLWithProperties},
		{pomXMLWithMultipleProperties},
		{pomXMLWithMultiplePropertiesRemainder},
	}

	for _, test := range tests {
		t.Run(test.xml, func(t *testing.T) {
			reader := strings.NewReader(test.xml)
			pomXML := ReadPomXMLFromFile(reader)
			if pomXML == nil {
				t.Fatalf("failed to read pom xml")
			}
			packageId := pomXML.GetPackageId()
			if packageId != "Maven|com.fasterxml.jackson:jackson-databind@2.13.1" {
				t.Fatalf("unexpected package id %s", packageId)
			}
		})
	}
}

func TestSealPomXML(t *testing.T) {
	tests := []struct {
		xml string
	}{
		{pomXMLParentGroupID},
		{pomXMLNoParent},
		{pomXMLParentVersion},
	}
	for _, test := range tests {
		t.Run(test.xml, func(t *testing.T) {
			reader := strings.NewReader(test.xml)
			pomXML := ReadPomXMLFromFile(reader)
			if pomXML == nil {
				t.Fatalf("failed to read pom xml")
			}
			if err := pomXML.Silence(); err != nil {
				t.Fatalf("failed to seal pom xml")
			}
			if pomXML.GetPackageId() != "Maven|seal.com.fasterxml.jackson:jackson-databind@2.13.1" {
				t.Fatalf("unexpected package id")
			}
		})
	}
}

func TestReadPomXMLFromFileInvalid(t *testing.T) {
	reader := strings.NewReader("invalid<><><><xml")
	pomXML := ReadPomXMLFromFile(reader)
	if pomXML != nil {
		t.Fatalf("should have failed to read pom xml")
	}
}

func TestPomXMLGetAsReader(t *testing.T) {
	pomXML := ReadPomXMLFromFile(strings.NewReader(pomXMLParentGroupID))
	reader := pomXML.GetAsReader()
	data, _ := io.ReadAll(reader)
	if string(data) != pomXMLParentGroupID {
		t.Fatalf("failed to get reader for pom xml")
	}
}

const pomXMLWithBadPropertiesInVersion = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${major</version>
<name>Spring Retry</name>
</project>`

const pomXMLWithBadPropertiesInVersion2 = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${maj${other}or}</version>
<name>Spring Retry</name>
<properties>
</properties>
</project>`

const pomXMLWithBadPropertiesNotExist = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${major}</version>
<name>Spring Retry</name>
</project>`

const pomXMLWithBadPropertyNotExist = `<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/maven-v4_0_0.xsd">
<groupId>com.fasterxml.jackson</groupId>
<artifactId>jackson-databind</artifactId>
<version>${major}</version>
<name>Spring Retry</name>
<properties>
</properties>
</project>`

func TestPomXMLResolveVersionFails(t *testing.T) {
	tests := []struct {
		xml string
	}{
		{pomXMLWithBadPropertiesInVersion},
		{pomXMLWithBadPropertiesInVersion2},
		{pomXMLWithBadPropertiesNotExist},
		{pomXMLWithBadPropertyNotExist},
	}

	for _, test := range tests {
		t.Run(test.xml, func(t *testing.T) {
			reader := strings.NewReader(test.xml)
			pomXML := ReadPomXMLFromFile(reader)
			if pomXML == nil {
				t.Fatalf("failed to read pom xml")
			}
			version := pomXML.GetVersion()
			if version != "" {
				t.Fatalf("unexpected package id %s", version)
			}
		})
	}
}
