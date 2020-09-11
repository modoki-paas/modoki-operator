package ghsink

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/pkg/config"
	"golang.org/x/sync/singleflight"
	"golang.org/x/xerrors"
)

// GitHubSink is a base to initialize GitHub App clients
type GitHubSink struct {
	appsTransport *ghinstallation.AppsTransport
	group         singleflight.Group
}

// NewGitHubSink initializes a utility to initialize GitHub App client
func NewGitHubSink(cfg *config.Config) (*GitHubSink, error) {
	tr := http.DefaultTransport
	gh := cfg.GitHub

	var appsTransport *ghinstallation.AppsTransport
	var err error

	if gh.PrivateKey.Path != "" {
		appsTransport, err = ghinstallation.NewAppsTransportKeyFromFile(tr, gh.AppID, gh.PrivateKey.Path)
	} else {
		appsTransport, err = ghinstallation.NewAppsTransport(tr, gh.AppID, []byte(gh.PrivateKey.Raw))
	}

	if err != nil {
		return nil, xerrors.Errorf("failed to initialize app transport: %w", err)
	}

	return &GitHubSink{
		appsTransport: appsTransport,
	}, nil
}

// AppsClient returns apps client(no specific installation)
func (ghs *GitHubSink) AppsClient() *github.Client {
	return github.NewClient(&http.Client{Transport: ghs.appsTransport})
}

// FindInstallationClient finds installation client from owner/repo
func (ghs *GitHubSink) FindInstallationClient(ctx context.Context, owner, repo string) (*github.Client, error) {
	ins, _, err := ghs.AppsClient().Apps.FindRepositoryInstallation(ctx, owner, repo)

	if err != nil {
		return nil, xerrors.Errorf("failed to find installation: %w", err)
	}

	return ghs.InstallationClient(ins.GetID()), nil
}

// InstallationClient returns API client for specific installation
func (ghs *GitHubSink) InstallationClient(installationID int64) *github.Client {
	key := strconv.FormatInt(installationID, 10)
	client, _, _ := ghs.group.Do(
		key,
		func() (interface{}, error) {
			go func() {
				time.After(14 * 24 * time.Hour)
				ghs.group.Forget(key)
			}()

			return ghs.installationClient(installationID), nil
		},
	)

	return client.(*github.Client)
}

func (ghs *GitHubSink) installationClient(installationID int64) *github.Client {
	itr := ghinstallation.NewFromAppsTransport(ghs.appsTransport, installationID)

	return github.NewClient(&http.Client{Transport: itr})
}

// InstallationToken returns token for specific installation
func (ghs *GitHubSink) InstallationToken(ctx context.Context, installationID int64) (string, error) {
	itr := ghinstallation.NewFromAppsTransport(ghs.appsTransport, installationID)

	token, err := itr.Token(ctx)

	if err != nil {
		return "", xerrors.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}
