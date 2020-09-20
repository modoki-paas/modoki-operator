package kpackbuilder

import (
	"context"
	"fmt"

	"github.com/modoki-paas/modoki-operator/pkg/k8sclientutil"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Status is a status of the generated resource
type Status int

const (
	NotExisting Status = iota
	Desired
	Undesirable
)

func (b *KpackBuilder) getServiceAccountName() string {
	return fmt.Sprintf("modoki-%s-builder", b.remoteSync.ObjectMeta.Name)
}

func (b *KpackBuilder) patchServiceAccount(sa *corev1.ServiceAccount, githubSecretName, imagePullSecretName string) (*corev1.ServiceAccount, error) {
	secretNames := []string{githubSecretName, imagePullSecretName}

	if sa != nil {
		expectedSecrets := map[string]struct{}{}

		for _, s := range secretNames {
			expectedSecrets[s] = struct{}{}
		}

		for _, s := range sa.Secrets {
			if githubSecretName == s.Name &&
				(s.Namespace == "" || s.Namespace == b.remoteSync.Namespace) {
				delete(expectedSecrets, s.Name)
			}
		}

		imagePullSecretFound := false

		for _, s := range sa.ImagePullSecrets {
			if imagePullSecretName == s.Name {
				imagePullSecretFound = true
				break
			}
		}

		if len(expectedSecrets) == 0 && imagePullSecretFound {
			return sa, nil
		}

	}

	var newSA *corev1.ServiceAccount
	if sa != nil {
		newSA = sa.DeepCopy()
	} else {
		newSA = &corev1.ServiceAccount{}
	}

	secrets := make([]corev1.ObjectReference, 0, len(secretNames))

	for i := range secretNames {
		secrets = append(secrets, corev1.ObjectReference{
			Name: secretNames[i],
		})
	}

	newSA.Name = b.getServiceAccountName()
	newSA.Namespace = b.remoteSync.Namespace
	newSA.Secrets = secrets

	if err := controllerutil.SetControllerReference(b.remoteSync, newSA, b.scheme); err != nil {
		return nil, xerrors.Errorf("failed to set ownerReferences to Image: %w", err)
	}

	return newSA, nil
}

func (b *KpackBuilder) findServiceAccount(ctx context.Context) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}

	if err := b.client.Get(ctx, client.ObjectKey{
		Name:      b.getServiceAccountName(),
		Namespace: b.remoteSync.Namespace,
	}, sa); err != nil {
		if errors.IsNotFound(err) {
			return nil, errNotFound
		}

		return nil, xerrors.Errorf("failed to find")
	}

	return sa, nil
}

func (b *KpackBuilder) prepareServiceAccount(ctx context.Context) (string, error) {
	githubSecretName := b.remoteSync.Spec.Base.GitHub.SecretName
	imagePullSecretName := b.remoteSync.Spec.Image.SecretName

	sa, err := b.findServiceAccount(ctx)

	switch err {
	case nil:
		newSA, err := b.patchServiceAccount(sa, githubSecretName, imagePullSecretName)

		if err != nil {
			return "", xerrors.Errorf("failed to get new ServiceAccount: %w", err)
		}

		if err := k8sclientutil.Patch(ctx, b.client, newSA, client.MergeFrom(sa)); err != nil {
			return "", xerrors.Errorf("failed to update existing ServiceAccount: %w", err)
		}
	case errNotFound:
		newSA, err := b.patchServiceAccount(nil, githubSecretName, imagePullSecretName)

		if err != nil {
			return "", xerrors.Errorf("failed to get new ServiceAccount: %w", err)
		}

		if err := b.client.Create(ctx, newSA); err != nil {
			return "", xerrors.Errorf("failed to create ServiceAccount: %w", err)
		}
	default:
		return "", xerrors.Errorf("failed to find ServiceAccount: %w", err)
	}

	return b.getServiceAccountName(), nil
}
