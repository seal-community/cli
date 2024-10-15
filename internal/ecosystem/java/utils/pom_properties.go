package utils

import (
	"bufio"
	"io"
	"strings"
)

type PomProperties struct {
	GroupId    string
	ArtifactId string
	Version    string
}

func ReadPomPropertiesFromFile(reader io.Reader) *PomProperties {
	pomProperties := &PomProperties{}
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "groupId") {
			pomProperties.GroupId = strings.Split(line, "=")[1]
		} else if strings.HasPrefix(line, "artifactId") {
			pomProperties.ArtifactId = strings.Split(line, "=")[1]
		} else if strings.HasPrefix(line, "version") {
			pomProperties.Version = strings.Split(line, "=")[1]
		}
	}

	if pomProperties.ArtifactId == "" || pomProperties.GroupId == "" || pomProperties.Version == "" {
		return nil
	}

	return pomProperties
}

func (p *PomProperties) GetAsReader() io.ReadCloser {
	properties := []string{
		"artifactId=" + p.ArtifactId,
		"groupId=" + p.GroupId,
		"version=" + p.Version,
	}
	return io.NopCloser(strings.NewReader(strings.Join(properties, "\n") + "\n"))
}

func (p *PomProperties) GetPackageId() string {
	return packageDependencyId(p.GroupId, p.ArtifactId, p.Version)
}
