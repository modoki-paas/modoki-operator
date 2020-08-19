package generators

import (
	"context"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Generator interface {
	Generate(ctx context.Context, app *v1alpha1.Application) ([]*unstructured.Unstructured, error)
}
