package mappings

// These need to be in a separate package outside of ecosystem, and api, so they can both rely on it
import (
	"log/slog"
)

// backend package manager enum
const (
	NpmManager    = "NPM"
	PythonManager = "PyPI"
	NugetManager  = "NuGet"
	MavenManger   = "Maven"
)

const (
	NodeEcosystem   = "node"
	PythonEcosystem = "python"
	DotnetEcosystem = ".NET"
	JavaEcosystem   = "java"
)

func BackendManagerToEcosystem(bem string) string {
	switch bem {
	case NpmManager:
		return NodeEcosystem
	case PythonManager:
		return PythonEcosystem
	case NugetManager:
		return DotnetEcosystem
	case MavenManger:
		return JavaEcosystem
	default:
		slog.Warn("unsupported manager", "manager", bem)
		return ""
	}
}

func EcosystemToBackendManager(es string) string {
	switch es {
	case NodeEcosystem:
		return NpmManager
	case PythonEcosystem:
		return PythonManager
	case DotnetEcosystem:
		return NugetManager
	case JavaEcosystem:
		return MavenManger
	default:
		slog.Warn("unsupported ecosystem", "value", es)
		return ""
	}
}
