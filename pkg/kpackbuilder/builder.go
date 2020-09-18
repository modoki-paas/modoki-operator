package kpackbuilder

import (
	"context"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/config"
	"golang.org/x/xerrors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errNotFound = xerrors.New("not found")
)

type KpackBuilder struct {
	client     client.Client
	remoteSync *v1alpha1.RemoteSync
	config     *config.Config
	scheme     *runtime.Scheme
}

func NewKpackBuilder(
	client client.Client,
	remoteSync *v1alpha1.RemoteSync,
	config *config.Config,
	scheme *runtime.Scheme,
) *KpackBuilder {
	return &KpackBuilder{
		client:     client,
		remoteSync: remoteSync,
		config:     config,
		scheme:     scheme,
	}
}

func (b *KpackBuilder) Run(ctx context.Context) error {
	saName, err := b.prepareServiceAccount(ctx)

	if err != nil {
		return xerrors.Errorf("failed to prepare ServiceAccount: %w", err)
	}

	imageName, err := b.prepareImage(ctx, saName)

	if err != nil {
		return xerrors.Errorf("failed to prepare Image: %w", err)
	}

	if len(imageName) == 0 {
		return nil
	}

	img := &v1alpha1.Application{}
	if err := b.client.Get(ctx, client.ObjectKey{
		Name:      b.remoteSync.Spec.ApplicationRef.Name,
		Namespace: b.remoteSync.Namespace,
	}, img); err != nil {
		return xerrors.Errorf("failed to get Application(%s): %w", b.remoteSync.Spec.ApplicationRef.Name, err)
	}

	if img.Spec.Image == imageName {
		return nil
	}

	newImg := img.DeepCopy()
	newImg.Spec.Image = imageName

	if err := b.client.Patch(ctx, newImg, client.MergeFrom(img)); err != nil {
		return xerrors.Errorf("failed to update image for Application: %w", err)
	}

	return nil
}
