package kpackbuilder

import (
	"context"

	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (b *KpackBuilder) getGitHubAccessToken(ctx context.Context, name, key string) (string, error) {
	secret := &corev1.Secret{}
	err := b.client.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: b.remoteSync.Namespace,
	}, secret)

	if err != nil {
		return "", xerrors.Errorf("failed to find secret: %w", err)
	}

	token, ok := secret.Data[key]

	if !ok {
		return "", xerrors.Errorf("unknown key: %s", key)
	}

	return string(token), nil
}
