package utils

import (
	"cli/internal/common"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

const projectTag = "project"
const groupIdTag = "groupId"
const artifactIdTag = "artifactId"
const versionTag = "version"
const parentTag = "parent"

type PomXML struct {
	Document etree.Document
}

var FailedSealingError = common.NewPrintableError("failed sealing pom.xml file")

func ReadPomXMLFromFile(reader io.Reader) *PomXML {
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(reader); err != nil {
		slog.Error("failed reading pom.xml", "err", err)
		return nil
	}
	return &PomXML{Document: *doc}
}

func (p *PomXML) GetGroupId() string {
	project := p.Document.SelectElement(projectTag)
	if project == nil {
		slog.Error("failed selecting project element")
		return ""
	}

	groupId := project.SelectElement(groupIdTag)
	if groupId != nil {
		return groupId.Text()
	}

	// if groupId is missing, it is assumed to be the parent's groupId
	parent := project.SelectElement(parentTag)
	if parent == nil {
		slog.Error("failed finding groupId element")
		return ""
	}

	groupId = parent.SelectElement(groupIdTag)
	if groupId == nil {
		slog.Error("failed selecting groupId element from parent")
		return ""
	}

	return groupId.Text()
}

func (p *PomXML) GetArtifactId() string {
	project := p.Document.SelectElement(projectTag)
	if project == nil {
		slog.Error("failed selecting project element")
		return ""
	}

	artifactId := project.SelectElement(artifactIdTag)
	if artifactId == nil {
		slog.Error("failed selecting artifactId element")
		return ""
	}

	return artifactId.Text()
}

func (p *PomXML) findProperty(project *etree.Element, propertyName string) string {
	// find the property in the properties tag
	slog.Debug("looking for property", "property", propertyName)
	properties := project.SelectElement("properties")
	if properties == nil {
		slog.Error("failed finding properties element")
		return ""
	}

	property := properties.SelectElement(strings.Trim(propertyName, "${}"))
	if property == nil {
		slog.Error("failed finding property element", "property", propertyName)
		return ""
	}

	return property.Text()
}

// Resolve version from pom.xml, supporting only properties
// https://maven.apache.org/pom.html#Properties
// Not parsing other expressions since it is not needed
// Not using `mvn help:evaluate` since this runs on each pom.xml and when looking for shaded dependencies, it's too slow
func (p *PomXML) resolveValue(value *etree.Element, project *etree.Element) string {
	valueText := value.Text()
	if !strings.Contains(valueText, "${") {
		slog.Debug("value does not contain a property", "value", valueText)
		return valueText
	}
	slog.Debug("resolving value", "value", valueText)
	re := regexp.MustCompile(`(?U)\$\{(.+)\}`)

	resolved := ""
	lastIdx := 0
	matches := re.FindAllStringSubmatchIndex(valueText, -1)

	if len(matches) == 0 {
		slog.Error("failed finding matches", "value", valueText)
		return ""
	}

	for _, match := range matches {
		exprStart := match[0]
		exprEnd := match[1]
		valueStart := match[2]
		valueEnd := match[3]
		resolved = resolved + valueText[lastIdx:exprStart]

		extracted := valueText[valueStart:valueEnd]

		propertyValue := p.findProperty(project, extracted)
		if propertyValue == "" {
			slog.Error("failed finding property", "propertyName", extracted, "value", propertyValue)
			return ""
		}

		resolved = resolved + propertyValue
		lastIdx = exprEnd
	}

	resolved = resolved + valueText[lastIdx:] // remainder

	slog.Debug("resolved value", "resolved", resolved)
	return resolved
}

func (p *PomXML) GetVersion() string {
	project := p.Document.SelectElement(projectTag)
	if project == nil {
		slog.Error("failed selecting project element")
		return ""
	}

	version := project.SelectElement(versionTag)
	if version != nil {
		return p.resolveValue(version, project)
	}

	// if version is missing, it is assumed to be the parent's version
	parent := project.SelectElement(parentTag)
	if parent == nil {
		slog.Error("failed finding version element")
		return ""
	}

	version = parent.SelectElement(versionTag)
	if version == nil {
		slog.Error("failed selecting version element from parent")
		return ""
	}

	return p.resolveValue(version, parent)
}

func (p *PomXML) GetPackageId() string {
	groupId := p.GetGroupId()
	artifactId := p.GetArtifactId()
	version := p.GetVersion()

	if groupId == "" || artifactId == "" || version == "" {
		slog.Error("failed getting packageId")
		return ""
	}

	return packageDependencyId(groupId, artifactId, version)
}

func (p *PomXML) GetAsReader() io.ReadCloser {
	s, err := p.Document.WriteToString()
	if err != nil {
		slog.Error("failed writing pom.xml", "err", err)
		return nil
	}
	return io.NopCloser(strings.NewReader(s))
}

func (p *PomXML) Silence() error {
	slog.Info("Changing groupId in pom.xml")

	project := p.Document.SelectElement(projectTag)
	if project == nil {
		slog.Error("failed selecting project element")
		return FailedSealingError
	}

	// groupId can be inherited from parent, but if it exists - update it
	groupId := project.SelectElement(groupIdTag)
	if groupId != nil {
		slog.Debug("groupId found")
		groupId.SetText(sealGroupId)
	}

	// not all pom.xml files have a parent tag
	// but if parent tag is present, and it has groupId, the groupId should be updated
	parent := project.SelectElement(parentTag)
	if parent != nil {
		slog.Debug("parent found")
		parentGroupId := parent.SelectElement(groupIdTag)
		if parentGroupId != nil {
			slog.Debug("parent groupId found")
			parentGroupId.SetText(sealGroupId)
		}
	}

	return nil
}

func (p *PomXML) GetPomProperties() *PomProperties {
	artifactId := p.GetArtifactId()
	groupId := p.GetGroupId()
	version := p.GetVersion()

	if artifactId == "" || groupId == "" || version == "" {
		slog.Error("failed getting pom properties", "artifactId", artifactId, "groupId", groupId, "version", version)
		return nil
	}

	return &PomProperties{
		ArtifactId: artifactId,
		GroupId:    groupId,
		Version:    version,
	}
}
