package mappings

// These need to be in a separate package outside of ecosystem, and api, so they can both rely on it
import (
	"log/slog"
)

// backend package manager enum
const (
	NpmManager      = "NPM"
	PythonManager   = "PyPI"
	NugetManager    = "NuGet"
	MavenManager    = "Maven"
	GolangManager   = "GO"
	ComposerManager = "Composer"
	RpmManager      = "RPM"
	DebManager      = "DEB"
)

const (
	NodeEcosystem   = "node"
	PythonEcosystem = "python"
	DotnetEcosystem = ".NET"
	JavaEcosystem   = "java"
	GolangEcosystem = "golang"
	PhpEcosystem    = "php"
	RpmEcosystem    = "rpm"
	DebEcosystem    = "deb"
)

func BackendManagerToEcosystem(bem string) string {
	switch bem {
	case NpmManager:
		return NodeEcosystem
	case PythonManager:
		return PythonEcosystem
	case NugetManager:
		return DotnetEcosystem
	case MavenManager:
		return JavaEcosystem
	case GolangManager:
		return GolangEcosystem
	case ComposerManager:
		return PhpEcosystem
	case RpmManager:
		return RpmEcosystem
	case DebManager:
		return DebEcosystem
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
		return MavenManager
	case GolangEcosystem:
		return GolangManager
	case PhpEcosystem:
		return ComposerManager
	case RpmEcosystem:
		return RpmManager
	case DebEcosystem:
		return DebManager
	default:
		slog.Warn("unsupported ecosystem", "value", es)
		return ""
	}
}
