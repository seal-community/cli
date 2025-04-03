package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"log/slog"
	"slices"
)

var includeArchPackageManagers = []string{mappings.ApkManager}

var hostArchToBackendArch = map[string]string{
	"amd64":  "x86_64",
	"x86_64": "x86_64",
	"armhf":  "arm64",
	"arm64":  "arm64",
	"arm":    "arm",
	"noarch": "any",
}

// creates the query to get the signatures for the packages from the backend
// includes architecture information for package managers that require it
// otherwise uses nil which means all architectures
// should be unique because of the filename, or use architecture if not
func createSignaturesQuery(packages []shared.PackageDownload) (api.ArtifactUniqueIdentifierList, error) {
	uids := make([]api.ArtifactUniqueIdentifier, 0, len(packages))
	for _, downloadedPackage := range packages {
		var arch string = ""
		if slices.Contains(includeArchPackageManagers, downloadedPackage.Entry.VulnerablePackage.Library.PackageManager) {
			if backendArch, ok := hostArchToBackendArch[downloadedPackage.Entry.Locations[""].Arch]; ok {
				arch = backendArch
			} else {
				slog.Error("failed to map host arch to backend arch", "hostArch", downloadedPackage.Entry.Locations[""].Arch)
				return api.ArtifactUniqueIdentifierList{}, fmt.Errorf("unsupported architecture %s", downloadedPackage.Entry.Locations[""].Arch)
			}
		}

		var archPtr *string = nil
		if arch != "" {
			archPtr = &arch
		}

		uids = append(uids, api.ArtifactUniqueIdentifier{
			LibraryVersionId: downloadedPackage.Entry.AvailableFix.VersionId,
			FileName:         downloadedPackage.ArtifactFileName,
			Architecture:     archPtr,
		})
	}

	return api.ArtifactUniqueIdentifierList{Entries: uids}, nil
}

func extractMessage(data []byte) string {
	sha := sha512.Sum512(data)
	return base64.StdEncoding.EncodeToString(sha[:])
}

type dataSignature struct {
	packageName string
	fileName    string
	data        []byte
	signature   string
}

// gets all the downloaded packages and the signatures from the backend
// and matches each one and returns a list of dataSignature which includes each package data and its signature
func matchPackageToSignature(packages []shared.PackageDownload, signatures []api.ArtifactMetadataResponse) ([]dataSignature, error) {
	result := make([]dataSignature, 0)
	signaturesMap := make(map[string][]api.ArtifactMetadataResponse)
	for _, signature := range signatures {
		signaturesMap[signature.FileName] = append(signaturesMap[signature.FileName], signature)
	}

	for _, downloadedPackage := range packages {
		if _, ok := signaturesMap[downloadedPackage.ArtifactFileName]; !ok {
			return nil, fmt.Errorf("Signature for package %s not found", downloadedPackage.Entry.VulnerablePackage.Descriptor())
		}

		for _, signature := range signaturesMap[downloadedPackage.ArtifactFileName] {
			if signature.LibraryVersionId != downloadedPackage.Entry.AvailableFix.VersionId {
				slog.Error("signature for package does not match", "package", downloadedPackage.Entry.VulnerablePackage.Descriptor())
				return nil, fmt.Errorf("Signature for package %s does not match", downloadedPackage.Entry.VulnerablePackage.Descriptor())
			}

			if slices.Contains(includeArchPackageManagers, downloadedPackage.Entry.VulnerablePackage.Library.PackageManager) {
				if *signature.Architecture != hostArchToBackendArch[downloadedPackage.Entry.Locations[""].Arch] {
					continue
				}
			}

			result = append(result, dataSignature{
				packageName: downloadedPackage.Entry.VulnerablePackage.Library.Name,
				fileName:    downloadedPackage.ArtifactFileName,
				data:        downloadedPackage.Data,
				signature:   signature.SealSignature,
			})
		}
	}

	if len(result) != len(packages) {
		slog.Error("some packages are missing signatures")
		return nil, fmt.Errorf("Some packages are missing signatures")
	}

	return result, nil
}

// validates the signatures of the downloaded packages using the seal signatures from the backend
func verifyPackagesSingatures(backend api.Backend, packages []shared.PackageDownload) error {
	// get the public key from the backend
	publicKeyBase64, err := backend.GetPublicKey()
	if err != nil {
		slog.Error("failed getting public key", "err", err)
		return fmt.Errorf("failed getting public key")
	}

	publicKey, err := common.LoadECDSAPublicKeyFromBase64(publicKeyBase64)
	if err != nil {
		slog.Error("failed loading public key", "err", err)
		return fmt.Errorf("failed loading public key")
	}

	// collect the artifacts and get the signatures
	query, err := createSignaturesQuery(packages)
	if err != nil {
		slog.Error("failed creating signatures query", "err", err)
		return err
	}

	signatures, err := backend.GetSignatures(&query)
	if err != nil {
		slog.Error("failed getting signatures", "err", err)
		return fmt.Errorf("failed getting signatures")
	}

	packageToSignature, err := matchPackageToSignature(packages, signatures)
	if err != nil {
		slog.Error("failed validating signatures", "err", err)
		return err
	}

	for _, toVerify := range packageToSignature {
		message := extractMessage(toVerify.data)
		valid, err := common.VerifySignature(publicKey, message, toVerify.signature)
		if err != nil {
			slog.Error("failed verifying signature", "err", err)
			return fmt.Errorf("failed verifying signature")
		}

		if !valid {
			slog.Error("signature for package is invalid", "package", toVerify.packageName, "filename", toVerify.fileName)
			return fmt.Errorf("Signature for package %s is invalid", toVerify.packageName)
		}
	}

	return nil
}
