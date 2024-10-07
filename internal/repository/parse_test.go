package repository

import (
	"fmt"
	"testing"

	"gopkg.in/src-d/go-git.v4/config"
)

func TestParseRemoteUrl(t *testing.T) {
	ps := map[string]string{
		"https://github.com/kensetsu/house.git":     "kensetsu/house",
		"https://127.0.01/kensetsu/house.git":       "kensetsu/house",
		"ssh://user@server:/kensetsu/house.git":     "kensetsu/house",
		"ssh://user@server:8080/kensetsu/house.git": "kensetsu/house",
		"ssh://git@github.com/kensetsu/house.git":   "kensetsu/house",
		"git@github.com:kensetsu/house.git":         "kensetsu/house",

		// gitlab
		"https://gitlab.com/kensetsu/house":  "kensetsu/house",
		"https://gitlab.com/kensetsu/house/": "kensetsu/house",

		// azure devops
		"https://myuser0123@dev.azure.com/myuser0123/myproj/_git/myrpeo": "myuser0123/myproj/_git/myrpeo", // IMPORTANT: the _git path element will be taken into consideration when generating project ids
	}

	i := 1
	for url, expected := range ps {
		t.Run(fmt.Sprintf("remote_%d", i), func(t *testing.T) {
			i++

			proj, err := GetProjectFromRemote(url)
			if err != nil {
				t.Fatalf("failed parsing: %v", err)
			}

			if proj != expected {
				t.Fatalf("bad proj %s. expected %s", proj, expected)
			}
		})
	}
}

func TestGetUrlForRemoteMultipleUrls(t *testing.T) {
	remoteName := "asd"
	remoteUrlA := "http://a.a"
	remoteUrlB := "http://b.b"
	remote := &config.RemoteConfig{Name: remoteName, URLs: []string{remoteUrlA, remoteUrlB}}

	url := getUrlForRemote(remote)
	if url != remoteUrlA {
		t.Fatalf("wrong url %s", url)
	}
}

func TestGetUrlForRemote(t *testing.T) {
	remoteName := "asd"
	remoteUrlB := "http://b.b"
	remote := &config.RemoteConfig{Name: remoteName, URLs: []string{remoteUrlB}}

	url := getUrlForRemote(remote)
	if url != remoteUrlB {
		t.Fatalf("wrong url %s", url)
	}
}

func TestGetUrlForRemoteNoUrls(t *testing.T) {
	remoteName := "asd"
	remote := &config.RemoteConfig{Name: remoteName}

	url := getUrlForRemote(remote)
	if url != "" {
		t.Fatalf("wrong url %s", url)
	}
}

func TestChooseRemoteNoOrigin(t *testing.T) {
	remoteNameA := "asd-a"
	remoteUrlA := "http://a.a"
	remoteA := &config.RemoteConfig{Name: remoteNameA, URLs: []string{remoteUrlA}}

	remoteNameB := "asd-b"
	remoteUrlB := "http://b.b"
	remoteB := &config.RemoteConfig{Name: remoteNameB, URLs: []string{remoteUrlB}}

	// uses sort on origin names
	remote := chooseFromRemotes(map[string]*config.RemoteConfig{remoteNameA: remoteA, remoteNameB: remoteB})

	if remote != remoteA {
		t.Fatalf("wrong remote %v", remote)
	}
}

func TestChooseRemoteOrigin(t *testing.T) {
	remoteNameA := "asd-a"
	remoteUrlA := "http://a.a"
	remoteA := &config.RemoteConfig{Name: remoteNameA, URLs: []string{remoteUrlA}}

	remoteNameB := "origin"
	remoteUrlB := "http://b.b"
	remoteB := &config.RemoteConfig{Name: remoteNameB, URLs: []string{remoteUrlB}}

	remote := chooseFromRemotes(map[string]*config.RemoteConfig{remoteNameA: remoteA, remoteNameB: remoteB})

	if remote != remoteB {
		t.Fatalf("wrong remote %v", remote)
	}
}

func TestChooseRemoteNone(t *testing.T) {
	remote := chooseFromRemotes(map[string]*config.RemoteConfig{})
	if remote != nil {
		t.Fatalf("wrong remote %v", remote)
	}
}
