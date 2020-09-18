package kpackbuilder

import (
	"context"
	"fmt"

	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return fmt.Sprintf("modoki-%s-%s", b.remoteSync.ObjectMeta.Name, b.remoteSync.Name)
}

func (b *KpackBuilder) newServiceAccount(secretNames []string) (*corev1.ServiceAccount, error) {
	secrets := make([]corev1.ObjectReference, 0, len(secretNames))

	for i := range secretNames {
		secrets = append(secrets, corev1.ObjectReference{
			Name: secretNames[i],
		})
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.getServiceAccountName(),
			Namespace: b.remoteSync.Namespace,
		},
		Secrets: secrets,
	}

	if err := controllerutil.SetControllerReference(b.remoteSync, sa, b.scheme); err != nil {
		return nil, xerrors.Errorf("failed to set ownerReferences to Image: %w", err)
	}

	return sa, nil
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
	secretNames := []string{
		b.remoteSync.Spec.Base.GitHub.SecretName,
		b.remoteSync.Spec.Image.SecretName,
	}

	sa, err := b.findServiceAccount(ctx)
	newSA, err := b.newServiceAccount(secretNames)

	if err != nil {
		return "", xerrors.Errorf("failed to get new ServiceAccount: %w", err)
	}

	switch err {
	case nil:
		if err := b.client.Patch(ctx, newSA, client.MergeFrom(sa)); err != nil {
			return "", xerrors.Errorf("failed to update existing ServiceAccount: %w", err)
		}
	case errNotFound:
		if err := b.client.Create(ctx, newSA); err != nil {
			return "", xerrors.Errorf("failed to create ServiceAccount: %w", err)
		}
	default:
		return "", xerrors.Errorf("failed to find ServiceAccount: %w", err)
	}

	return b.getServiceAccountName(), nil
}
