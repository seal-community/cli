package utils

import (
	"cli/internal/common"
	"io"
	"log/slog"
	"strings"

	"github.com/beevik/etree"
)

const projectTag = "project"
const groupIdTag = "groupId"
const artifactIdTag = "artifactId"
const versionTag = "version"
const parentTag = "parent"

type pomXML struct {
	Document etree.Document
}

var FailedSealingError = common.NewPrintableError("failed sealing pom.xml file")

func ReadPomXMLFromFile(reader io.Reader) *pomXML {
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(reader); err != nil {
		slog.Error("failed reading pom.xml", "err", err)
		return nil
	}
	return &pomXML{Document: *doc}
}

func (p *pomXML) GetGroupId() string {
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

func (p *pomXML) GetArtifactId() string {
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

func (p *pomXML) GetVersion() string {
	project := p.Document.SelectElement(projectTag)
	if project == nil {
		slog.Error("failed selecting project element")
		return ""
	}

	version := project.SelectElement(versionTag)
	if version != nil {
		return version.Text()
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

	return version.Text()
}

func (p *pomXML) GetPackageId() string {
	groupId := p.GetGroupId()
	artifactId := p.GetArtifactId()
	version := p.GetVersion()

	if groupId == "" || artifactId == "" || version == "" {
		slog.Error("failed getting packageId")
		return ""
	}

	return packageDependencyId(groupId, artifactId, version)
}

func (p *pomXML) GetAsReader() io.ReadCloser {
	s, err := p.Document.WriteToString()
	if err != nil {
		slog.Error("failed writing pom.xml", "err", err)
		return nil
	}
	return io.NopCloser(strings.NewReader(s))
}

func (p *pomXML) Silence() error {
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

func (p *pomXML) GetPomProperties() *PomProperties {
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
