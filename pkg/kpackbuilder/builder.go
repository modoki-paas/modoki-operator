package kpackbuilder

import (
	"fmt"

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/config"
	kpacktypes "github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type KpackBuilder struct {
}

func (b *KpackBuilder) Generate(cfg *config.Config, remoteSync *v1alpha1.RemoteSync) ([]runtime.Object, error) {
	spec := remoteSync.Spec

	githubSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"kpack.io/git": cfg.GitHub.URL,
			},
			Name: "github-secret",
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			corev1.BasicAuthUsernameKey: "x-access-token",
			corev1.BasicAuthPasswordKey: "",
		},
	}

	serviceAccount := &corev1.ServiceAccount{
		Secrets: []corev1.ObjectReference{
			{
				Name: "",
			},
		},
	}

	img := &kpacktypes.Image{
		Spec: kpacktypes.ImageSpec{
			ServiceAccount: serviceAccount.ObjectMeta.Name,
			Builder:        cfg.Builder,
			Source: kpacktypes.SourceConfig{
				Git: &kpacktypes.Git{
					URL:      fmt.Sprintf("%s/%s/%s", cfg.GitHub.URL, spec.Base.GitHub.Owner, spec.Base.GitHub.Repository),
					Revision: spec.Base.GitHub.Branch,
				},
				SubPath: remoteSync.Spec.Base.SubPath,
			},
			FailedBuildHistoryLimit:  int64Ptr(3),
			SuccessBuildHistoryLimit: int64Ptr(5),
		},
	}

	return []runtime.Object{
		githubSecret,
		serviceAccount,
		img,
	}, nil
}

func int64Ptr(v int64) *int64 {
	return &v
}
