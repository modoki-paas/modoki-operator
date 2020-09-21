package k8sclientutil

import (
	"context"

	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetGitHubAccessToken(ctx context.Context, c client.Client, name, namespace, key string) (string, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
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
