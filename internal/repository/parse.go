package repository

import (
	"errors"
	"log/slog"
	"sort"
	"strings"

	"github.com/whilp/git-urls"
	"golang.org/x/exp/maps"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
)

func getGitRemotesForDir(dir string) (map[string]*config.RemoteConfig, error) {
	// will recrusively go up and search for the .git dir
	r, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true})
	if err == git.ErrRepositoryNotExists {
		slog.Info("not within a repository")
		return nil, nil // not considered an error
	}

	if err != nil {
		slog.Error("failed looking up repository", "err", err, "path", dir)
		return nil, err
	}

	wt, err := r.Worktree()
	if err != nil {
		slog.Error("failed looking up repository worktree", "err", err)
		return nil, err
	}

	if wt.Filesystem == nil {
		slog.Error("empty filesystem on repository working tree")
		return nil, errors.New("empty worktree")
	}

	slog.Info("discovered git repository", "path", wt.Filesystem.Root())
	c, err := r.Config()
	if err != nil {
		slog.Error("failed getting repository config", "err", err)
		return nil, err
	}

	return c.Remotes, nil
}

func getUrlForRemote(remote *config.RemoteConfig) string {
	if len(remote.URLs) == 0 {
		slog.Warn("no urls for remote", "name", remote.Name)
		return ""
	}

	if len(remote.URLs) > 1 {
		slog.Warn("multiple remote urls for remote", "name", remote.Name, "urls", remote.URLs)
	}

	// chossing first url
	return remote.URLs[0]
}

func chooseFromRemotes(remotes map[string]*config.RemoteConfig) *config.RemoteConfig {
	// prefer origin, as it is the default remote name most of the times
	if remote, ok := remotes["origin"]; ok {
		return remote
	}

	// choosing the first remote, using sort since map is unordered
	originNames := maps.Keys(remotes)
	sort.Strings(originNames)

	if len(originNames) == 0 {
		slog.Warn("no remotes found for repo")
		return nil
	}

	origin := originNames[0]
	slog.Warn("origin remote not found; choosing first", "origin", origin, "remotes", remotes)
	return remotes[origin]
}

// will traverse up the tree until the first repository is found, similar to how git works
// returns the remote url for the repo, if found, otherwise empty string
func FindGitRemoteUrl(dir string) (string, error) {

	remotes, err := getGitRemotesForDir(dir)
	if len(remotes) == 0 {
		// either err or not a git dir, err could be nil
		return "", err
	}

	if len(remotes) > 1 {
		slog.Warn("found multiple remotes configured for repository", "count", len(remotes))
	}

	remote := chooseFromRemotes(remotes)
	if remote == nil {
		slog.Info("no remotes found")
		return "", nil
	}

	return getUrlForRemote(remote), nil
}

func GetProjectFromRemote(remote string) (string, error) {
	remote = strings.TrimSpace(remote)
	// net/url parser does not like newlines. We switched to git-urls but wanted to be compatible
	if strings.Contains(remote, "\n") {
		return "", errors.New("newline in remote url")
	}
	parsed, err := giturls.Parse(remote)
	if err != nil {
		return "", err
	}

	proj := parsed.Path
	proj = strings.TrimSuffix(proj, ".git") // not always present
	proj = strings.Trim(proj, "/")          // clean just in case

	return proj, nil
}
