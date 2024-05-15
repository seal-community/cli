package shared

import (
	"io"
	"log/slog"

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

func SavePackageMetadata(metadata SealPackageMetadata, w io.Writer) error {
	yamlEncoder := yaml.NewEncoder(w)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(metadata)
	if err != nil {
		return err
	}

	return nil
}
