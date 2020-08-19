package generators_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/generators"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCommandGenerator(t *testing.T) {
	generator := generators.NewCommandGenerator("node", "main.js")

	generator.SetWorkingDirectory("../cdk8s-template")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app := &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "modoki.tsuzu.dev/v1alpha1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "application-sample2",
			Namespace: "modoki-app",
		},
		Spec: v1alpha1.ApplicationSpec{
			Domains: []string{"example.tsuzu.dev"},
			Image:   "httpd:latest",
			Command: []string{"/usr/bin/httpd"},
			Args:    []string{"-D"},
			Attributes: map[string]string{
				"foo":  "bar",
				"hoge": "fuga",
			},
		},
	}

	objs, err := generator.Generate(ctx, app)

	if err != nil {
		t.Fatalf("failed to generate: %+v", err)
	}

	for i := range objs {
		fmt.Println(objs[i].GroupVersionKind())
	}
}
