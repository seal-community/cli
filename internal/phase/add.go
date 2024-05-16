package phase

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"errors"
	"log/slog"
)

const AddSteps = 2

type addPhase struct {
	*scanPhase
}

func NewAddPhase(projectDir string, configPath string, showProgress bool) (*addPhase, error) {
	sp, err := NewScanPhase(projectDir, configPath, showProgress)
	if err != nil {
		return nil, err
	}

	sp.Bar.ChangeMax(AddSteps) // only exposing the add phase steps

	ap := &addPhase{
		scanPhase: sp,
	}

	return ap, nil
}

// the requested from/to pair
type AddRule struct {
	From actions.Override
	To   *actions.Override // using pointer here to differentiate between cases, as it's optional
}

func (r AddRule) isSafest() bool {
	// should instruct the CLI to always fetch the newest SP
	return r.To == nil
}

func (r AddRule) isLatest() bool {
	// should inform us to grab the latest SP we have and store it in the actions file
	return !r.isSafest() && (r.To.Library == "" || r.To.Version == "")
}

type ResolvedRule struct {
	From api.PackageVersion
	To   *api.PackageVersion // using pointer here to differentiate between cases
}

var overrideNotFound = errors.New("override not found")
var overrideMultipleCandidates = errors.New("multiple candidates found")

func (ap *addPhase) resolveOverride(manager string, o actions.Override, qt api.PackageQueryType) (*api.PackageVersion, error) {

	dep := common.Dependency{Name: o.Library, Version: o.Version, PackageManager: manager}
	result, err := ap.Server.FetchPackagesInfo([]common.Dependency{dep}, nil, qt, nil)
	if err != nil || result == nil {
		slog.Error("ƒailed querying package", "err", err, "from-library", o.Library, "from-version", o.Version)
		return nil, err
	}
	resolvedList := *result

	if len(resolvedList) > 1 {
		slog.Warn("got multiple results for package", "package", dep, "results", len(resolvedList))
		return nil, overrideMultipleCandidates // should not happen
	}

	if len(resolvedList) == 0 {
		return nil, overrideNotFound
	}

	return &(resolvedList[0]), nil
}

func (ap *addPhase) Resolve(rule AddRule) (*ResolvedRule, error) {
	slog.Info("starting rule resolution", "target", ap.ProjectDir)

	ap.Bar.Describe("Checking package version")

	mngr := mappings.EcosystemToBackendManager(ap.Manager.GetEcosystem())

	if mngr == "" {
		return nil, common.NewPrintableError("unsupported package manager for ecosystem: %s", ap.Manager.GetEcosystem())
	}

	var resolvedTo *api.PackageVersion
	resolvedFrom, err := ap.resolveOverride(mngr, rule.From, api.OnlyVulnerable)

	if err == overrideNotFound || err == overrideMultipleCandidates {
		// we want to print this to the user specifically, to tell that the input is bad
		slog.Warn("did not find the origin version", "from-library", rule.From.Library, "from-version", rule.From.Version)
		return nil, common.NewPrintableError("could not find version %s %s", common.Colorize(rule.From.Library, common.AnsiDarkGrey), common.Colorize(rule.From.Version, common.AnsiDarkGrey)) // will be shown to user
	} else if err != nil {
		slog.Error("did not find the origin version", "err", err, "from-library", rule.From.Library, "from-version", rule.From.Version)
		return nil, common.WrapWithPrintable(err, "could not find version")
	}

	ap.advanceStep("Looking for a fix")
	if !rule.isSafest() && !rule.isLatest() {
		// rule To is not nil and has content
		slog.Debug("resolving using provided To values", "library", rule.To.Library, "version", rule.To.Version)
		resolvedTo, err = ap.resolveOverride(mngr, *rule.To, api.OnlyFixed)
	} else {
		// get the recommended version from the resolved 'From'
		// NOTE: in the future we should add support to safest here
		slog.Debug("resolving using the remote recommended value", "version", resolvedFrom.RecommendedLibraryVersionString)
		resolvedTo, err = ap.resolveOverride(mngr, actions.Override{
			Library: resolvedFrom.Library.Name,
			Version: resolvedFrom.RecommendedLibraryVersionString,
		}, api.OnlyFixed)
	}

	if err == overrideNotFound || err == overrideMultipleCandidates {
		// we want to print this to the user specifically, to tell that the input is bad
		slog.Warn("did not find the target version", "from-library", rule.From.Library, "from-version", rule.From.Version)
		return nil, common.NewPrintableError("sealed version not found for %s %s", common.Colorize(rule.From.Library, common.AnsiDarkGrey), common.Colorize(rule.From.Version, common.AnsiDarkGrey)) // will be shown to user
	} else if err != nil {
		slog.Error("failed querying target version")
		return nil, common.FallbackPrintableMsg(err, "failed resolving version")
	}

	ap.advanceStep("") // final step

	return &ResolvedRule{From: *resolvedFrom, To: resolvedTo}, nil
}
