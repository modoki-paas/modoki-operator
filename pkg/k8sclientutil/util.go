package k8sclientutil

import (
	"context"

	"golang.org/x/xerrors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Patch(ctx context.Context, client client.Client, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	diff, err := patch.Data(obj)

	if err != nil {
		return xerrors.Errorf("failed to calc patch: %w", err)
	}

	if len(diff) <= 2 {
		return nil
	}

	if err := client.Patch(ctx, obj, patch, opts...); err != nil {
		return xerrors.Errorf("failed to patch: %w", err)
	}

	return nil
}
