package kpackbuilder

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/pkg/tokentransport"
	kpacktypes "github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func (b *KpackBuilder) getKpackImageName() string {
	return fmt.Sprintf("modoki-%s-%s", b.remoteSync.ObjectMeta.Name, b.remoteSync.Name)
}

func (b *KpackBuilder) newImage(saName, revision string) (*kpacktypes.Image, error) {
	u, err := url.Parse(b.config.GitHub.URL)

	if err != nil {
		return nil, xerrors.Errorf("failed to parse GitHub url: %w", err)
	}

	spec := b.remoteSync.Spec
	u.Path = path.Join(spec.Base.GitHub.Owner, spec.Base.GitHub.Repository)

	img := &kpacktypes.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.getKpackImageName(),
			Namespace: b.remoteSync.Namespace,
		},
		Spec: kpacktypes.ImageSpec{
			ServiceAccount: saName,
			Builder:        b.config.Builder,
			Source: kpacktypes.SourceConfig{
				Git: &kpacktypes.Git{
					URL:      u.String(),
					Revision: revision,
				},
				SubPath: spec.Base.SubPath,
			},
			FailedBuildHistoryLimit:  int64Ptr(3),
			SuccessBuildHistoryLimit: int64Ptr(5),
			Build: &kpacktypes.ImageBuild{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("500m"),
					},
				},
			},
		},
	}

	if err := controllerutil.SetOwnerReference(b.remoteSync, img, b.scheme); err != nil {
		return nil, xerrors.Errorf("failed to set ownerReferences to Image: %w", err)
	}

	return img, nil
}

func (b *KpackBuilder) findImage(ctx context.Context) (*kpacktypes.Image, error) {
	image := &kpacktypes.Image{}

	err := b.client.Get(ctx, client.ObjectKey{
		Name:      b.getKpackImageName(),
		Namespace: b.remoteSync.Namespace,
	}, image)

	if errors.IsNotFound(err) {
		return nil, errNotFound
	}

	if err != nil {
		return nil, xerrors.Errorf("failed to find image: %w", err)
	}

	return image, nil
}

func (b *KpackBuilder) prepareImage(ctx context.Context, saName string) (string, error) {
	spec := b.remoteSync.Spec
	gh := spec.Base.GitHub
	secretName := gh.SecretName

	token, err := b.getGitHubAccessToken(ctx, secretName, "password")

	if err != nil {
		return "", xerrors.Errorf("failed to get access token from secret(%s): %w", secretName, err)
	}

	ghclient := github.NewClient(&http.Client{
		Transport: tokentransport.New(token),
	})

	var revision string
	switch {
	case len(gh.SHA) != 0:
		revision = gh.SHA
	case gh.PullRequest != nil:
		pr, _, err := ghclient.PullRequests.Get(ctx, gh.Owner, gh.Repository, *gh.PullRequest)

		if err != nil {
			return "", xerrors.Errorf("failed to get branch for %s: %w", gh.Branch, err)
		}
		revision = pr.GetMergeCommitSHA()
	default:
		b := gh.Branch
		if len(b) == 0 {
			b = "master"
		}

		branch, _, err := ghclient.Repositories.GetBranch(ctx, gh.Owner, gh.Repository, b)

		if err != nil {
			return "", xerrors.Errorf("failed to get branch for %s: %w", gh.Branch, err)
		}
		revision = branch.GetCommit().GetSHA()
	}

	image, err := b.findImage(ctx)
	newImage, err := b.newImage(saName, revision)

	if err != nil {
		return "", xerrors.Errorf("failed to prepare image: %w", err)
	}

	switch err {
	case nil:
		if err := b.client.Patch(ctx, newImage, client.MergeFrom(image)); err != nil {
			return "", xerrors.Errorf("failed to update existing Image: %w", err)
		}
	case errNotFound:
		if err := b.client.Create(ctx, newImage); err != nil {
			return "", xerrors.Errorf("failed to create Image: %w", err)
		}
	default:
		return "", xerrors.Errorf("failed to find Image: %w", err)
	}

	return image.Status.LatestImage, nil
}
