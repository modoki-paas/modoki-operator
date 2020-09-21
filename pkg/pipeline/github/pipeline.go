package ghpipeline

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/config"
	"github.com/modoki-paas/modoki-operator/pkg/k8sclientutil"
	"github.com/modoki-paas/modoki-operator/pkg/tokentransport"
	"golang.org/x/xerrors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GitHubPipeline struct {
	client   client.Client
	pipeline *v1alpha1.AppPipeline
	config   *config.Config
	scheme   *runtime.Scheme
	logger   logr.Logger
}

func NewGitHubPipeline(
	client client.Client,
	pipeline *v1alpha1.AppPipeline,
	config *config.Config,
	scheme *runtime.Scheme,
	logger logr.Logger,
) *GitHubPipeline {
	return &GitHubPipeline{
		client:   client,
		pipeline: pipeline,
		config:   config,
		scheme:   scheme,
		logger:   logger,
	}
}

func (p *GitHubPipeline) Run(ctx context.Context) error {
	gh := p.pipeline.Spec.Base.GitHub

	secretName := gh.SecretName
	token, err := k8sclientutil.GetGitHubAccessToken(ctx, p.client, secretName, p.pipeline.Namespace, "password")

	if err != nil {
		return xerrors.Errorf("failed to get access token from secret(%s): %w", secretName, err)
	}

	ghclient := github.NewClient(&http.Client{
		Transport: tokentransport.New(token),
	})

	list, _, err := ghclient.PullRequests.List(ctx, gh.Owner, gh.Repository, &github.PullRequestListOptions{
		State: "open",
	})

	if err != nil {
		return xerrors.Errorf("failed to get the list of pull requests: %w", err)
	}

	if err := p.prepareApplications(ctx, list); err != nil {
		return xerrors.Errorf("failed to prepare Applications: %w", err)
	}

	if err := p.prepareRemoteSyncs(ctx, list); err != nil {
		return xerrors.Errorf("failed to prepare RemoteSyncs: %w", err)
	}

	return nil
}
