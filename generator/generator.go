package generator

import (
	"context"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Generator interface {
	Generate(ctx context.Context, spec *v1alpha1.ApplicationSpec) ([]*unstructured.Unstructured, error)
}
