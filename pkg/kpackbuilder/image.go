package kpackbuilder

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/pkg/k8sclientutil"
	"github.com/modoki-paas/modoki-operator/pkg/tokentransport"
	kpacktypes "github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func shortSHA(revision string) string {
	var short string
	if len(revision) < 7 {
		short = revision
	} else {
		short = revision[:7]
	}

	return short
}

const (
	remoteSyncKey = "modoki.tsuzu.dev/remote-sync"
)

var (
	errNoAvailableRevision = xerrors.Errorf("unknown revision")
)

func (b *KpackBuilder) getKpackImageName(revision string) string {
	return fmt.Sprintf("modoki-%s-%s", b.remoteSync.ObjectMeta.Name, shortSHA(revision))
}

func (b *KpackBuilder) patchImage(image *kpacktypes.Image, saName, revision string) (*kpacktypes.Image, error) {
	u, err := url.Parse(b.config.GitHub.URL)

	if err != nil {
		return nil, xerrors.Errorf("failed to parse GitHub url: %w", err)
	}

	spec := b.remoteSync.Spec
	u.Path = path.Join(spec.Base.GitHub.Owner, spec.Base.GitHub.Repository)

	var newImage *kpacktypes.Image
	if image != nil {
		newImage = image.DeepCopy()
	} else {
		newImage = &kpacktypes.Image{}
	}

	newImage.Name = b.getKpackImageName(revision)
	newImage.Namespace = b.remoteSync.Namespace

	if newImage.Labels == nil {
		newImage.Labels = map[string]string{}
	}
	newImage.Labels[remoteSyncKey] = b.remoteSync.Name

	newImage.Spec.Builder = b.config.Builder
	newImage.Spec.ServiceAccount = saName
	newImage.Spec.Builder = b.config.Builder
	newImage.Spec.Source.Git = &kpacktypes.Git{
		URL:      u.String(),
		Revision: revision,
	}
	newImage.Spec.Source.SubPath = spec.Base.SubPath

	newImage.Spec.FailedBuildHistoryLimit = int64Ptr(3)
	newImage.Spec.SuccessBuildHistoryLimit = int64Ptr(5)
	newImage.Spec.Build = &kpacktypes.ImageBuild{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		},
	}

	newImage.Spec.Tag = fmt.Sprintf("%s:%s", spec.Image.Name, shortSHA(revision))

	if err := controllerutil.SetControllerReference(b.remoteSync, newImage, b.scheme); err != nil {
		return nil, xerrors.Errorf("failed to set ownerReferences to Image: %w", err)
	}

	return newImage, nil
}

func (b *KpackBuilder) findImage(ctx context.Context, revision string) (*kpacktypes.Image, error) {
	image := &kpacktypes.Image{}

	err := b.client.Get(ctx, client.ObjectKey{
		Name:      b.getKpackImageName(revision),
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

func (b *KpackBuilder) cleanupOldImages(ctx context.Context, revision string) error {
	images := &kpacktypes.ImageList{}
	err := b.client.List(ctx, images, client.MatchingLabels{
		remoteSyncKey: b.remoteSync.Name,
	}, client.InNamespace(b.remoteSync.Namespace))

	if err != nil {
		return xerrors.Errorf("failed to get the list of RemoteSync: %w", err)
	}

	errors := []string{}
	for i := range images.Items {
		if images.Items[i].Name != b.getKpackImageName(revision) {
			if err := b.client.Delete(ctx, &images.Items[i]); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	if len(errors) != 0 {
		return xerrors.Errorf("failed to delete old images: %s", strings.Join(errors, ","))
	}

	return nil
}

func (b *KpackBuilder) prepareImage(ctx context.Context, saName string) (string, error) {
	spec := b.remoteSync.Spec
	gh := spec.Base.GitHub
	secretName := gh.SecretName

	token, err := k8sclientutil.GetGitHubAccessToken(ctx, b.client, secretName, b.remoteSync.Namespace, "password")

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
		pr, _, err := ghclient.PullRequests.Get(ctx, gh.Owner, gh.Repository, int(*gh.PullRequest))

		if err != nil {
			return "", xerrors.Errorf("failed to get branch for PR(%d): %w", *gh.PullRequest, err)
		}
		revision = pr.GetMergeCommitSHA()
	default:
		b := gh.Branch
		if len(b) == 0 {
			b = "master"
		}

		branch, _, err := ghclient.Repositories.GetBranch(ctx, gh.Owner, gh.Repository, b)

		if err != nil {
			return "", xerrors.Errorf("failed to get branch for %s: %w", b, err)
		}
		revision = branch.GetCommit().GetSHA()
	}

	if len(revision) == 0 {
		return "", errNoAvailableRevision
	}

	if err := b.cleanupOldImages(ctx, revision); err != nil {
		return "", xerrors.Errorf("failed to cleanup old images: %w", err)
	}

	image, err := b.findImage(ctx, revision)

	switch err {
	case nil:
		newImage, err := b.patchImage(image, saName, revision)

		if err != nil {
			return "", xerrors.Errorf("failed to patch Image: %w", err)
		}

		if err := k8sclientutil.Patch(ctx, b.client, newImage, client.MergeFrom(image)); err != nil {
			return "", xerrors.Errorf("failed to update existing Image: %w", err)
		}
	case errNotFound:
		newImage, err := b.patchImage(nil, saName, revision)

		if err != nil {
			return "", xerrors.Errorf("failed to create Image: %w", err)
		}

		if err := b.client.Create(ctx, newImage); err != nil {
			return "", xerrors.Errorf("failed to create Image: %w", err)
		}
	default:
		return "", xerrors.Errorf("failed to find Image: %w", err)
	}

	latestImage := ""
	if image != nil {
		latestImage = image.Status.LatestImage
	}

	return latestImage, nil
}
