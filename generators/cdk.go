package generators

import (
	"context"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CDKGenerator struct {
}

var _ Generator = &CDKGenerator{}

func (g *CDKGenerator) Generate(ctx context.Context, spec *v1alpha1.ApplicationSpec) ([]*unstructured.Unstructured, error) {
	panic("not implemented")
}
