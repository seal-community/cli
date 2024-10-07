package shared

import (
	"fmt"
	"io"
	"log/slog"

	"cli/internal/common"

	"gopkg.in/yaml.v3"
)

const SealMetadataFileName = ".seal-metadata.yaml"

type SealPackageMetadata struct {
	SealedVersion string `yaml:"version"`
}

func LoadPackageMetadata(r io.Reader) (*SealPackageMetadata, error) {
	d := yaml.NewDecoder(r)
	d.KnownFields(true)
	metadata := &SealPackageMetadata{}

	if err := d.Decode(&metadata); err != nil {
		slog.Error("failed decoding yaml package metadata", "err", err)
		return nil, err
	}

	return metadata, nil
}

func LoadPackageSealMetadata(metadataFilePath string) (*SealPackageMetadata, error) {
	exists, err := common.PathExists(metadataFilePath)
	if err != nil {
		slog.Error("failed checking package metadata exists", "err", err)
		return nil, err
	}

	if !exists {
		slog.Debug("no metadata file found", "path", metadataFilePath)
		return nil, nil
	}

	slog.Info("loading metadata file", "path", metadataFilePath)
	r, err := common.OpenFile(metadataFilePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	metadata, err := LoadPackageMetadata(r)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func SavePackageMetadata(metadata SealPackageMetadata, metadataFilePath string) error {
	slog.Info("creating metadata file", "path", metadataFilePath)
	w, err := common.CreateFile(metadataFilePath)
	if err != nil {
		slog.Error("failed creating metadata file", "err", err)
		return err
	}

	return WritePackageMetadata(metadata, w)
}

func WritePackageMetadata(metadata SealPackageMetadata, w io.Writer) error {
	yamlEncoder := yaml.NewEncoder(w)
	yamlEncoder.SetIndent(2)

	err := yamlEncoder.Encode(metadata)
	if err != nil {
		slog.Error("failed saving metadata file", "err", err)
		return fmt.Errorf("failed saving metadata with error: %w", err)
	}

	return nil
}
