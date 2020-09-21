package ghpipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"golang.org/x/xerrors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (p *GitHubPipeline) mutateRemoteSync(rs *v1alpha1.RemoteSync, pr int) error {
	if rs.Labels == nil {
		rs.Labels = map[string]string{}
	}

	rs.Labels[appPipelineLabel] = p.pipeline.Name
	rs.Labels[pullReqIDLabel] = strconv.Itoa(pr)

	spec := &rs.Spec

	spec.ApplicationRef.Name = p.getAppName(pr)

	base := p.pipeline.Spec.Base
	spec.Base.GitHub.Owner = base.GitHub.Owner
	spec.Base.GitHub.Repository = base.GitHub.Repository
	spec.Base.GitHub.SecretName = base.GitHub.SecretName
	spec.Base.GitHub.PullRequest = &pr

	spec.Image.Name = p.pipeline.Spec.Image.Name
	spec.Image.SecretName = p.pipeline.Spec.Image.SecretName

	if err := ctrl.SetControllerReference(p.pipeline, rs, p.scheme); err != nil {
		return xerrors.Errorf("failed to set controller reference: %w", err)
	}

	return nil
}

func (p *GitHubPipeline) deleteObsoleteRemoteSyncs(ctx context.Context, prs []*github.PullRequest) error {
	apps := &v1alpha1.RemoteSyncList{}
	err := p.client.List(ctx, apps, client.MatchingLabels{
		appPipelineLabel: p.pipeline.Name,
	}, client.InNamespace(p.pipeline.Namespace))

	if err != nil {
		return xerrors.Errorf("failed to get the list of RemoteSync: %w", err)
	}

	ids := map[int]struct{}{}
	for i := range prs {
		ids[prs[i].GetNumber()] = struct{}{}
	}

	deleted := make([]string, 0, 5)
	for i := range apps.Items {
		id, err := strconv.Atoi(apps.Items[i].Labels[pullReqIDLabel])

		if err != nil {
			deleted = append(deleted, apps.Items[i].Name)

			continue
		}

		_, ok := ids[id]

		if !ok {
			deleted = append(deleted, apps.Items[i].Name)
		}
	}

	errors := []string{}
	for _, name := range deleted {
		err := p.client.Delete(ctx, &v1alpha1.RemoteSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: p.pipeline.Namespace,
			},
		})

		if err != nil && !apierrors.IsNotFound(err) {
			logger := p.logger.WithValues(
				"kind", "RemoteSync",
				"name", name,
				"namespace", p.pipeline.Namespace,
			)

			logger.Error(err, "failed to delete RemoteSync")

			errors = append(errors, err.Error())
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("Deleting RemoteSync failed: %s", strings.Join(errors, ","))
	}

	return nil
}

func (p *GitHubPipeline) prepareRemoteSyncs(ctx context.Context, prs []*github.PullRequest) error {
	for _, pr := range prs {
		id := pr.GetNumber()

		app := &v1alpha1.RemoteSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("modoki-pipeline-%s-pr-%d", p.pipeline.Name, id),
				Namespace: p.pipeline.Namespace,
			},
		}

		logger := p.logger.WithValues(
			"kind", "RemoteSync",
			"name", app.Name,
			"namespace", app.Namespace,
		)

		if res, err := ctrl.CreateOrUpdate(ctx, p.client, app, func() error {
			return p.mutateRemoteSync(app, id)
		}); err != nil {
			return xerrors.Errorf("failed to create/update RemoteSync(PR: %d): %w", id, err)
		} else {
			logger.Info("The RemoteSync operation succeeded", "op", res)
		}
	}

	if err := p.deleteObsoleteRemoteSyncs(ctx, prs); err != nil {
		return xerrors.Errorf("failed to clean up obsolete RemoteSyncs: %w", err)
	}

	return nil
}
