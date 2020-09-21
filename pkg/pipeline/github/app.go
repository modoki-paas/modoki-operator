package ghpipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"golang.org/x/xerrors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (p *GitHubPipeline) getAppName(pr int) string {
	return fmt.Sprintf("modoki-pipeline-%s-pr-%d", p.pipeline.Name, pr)
}

func (p *GitHubPipeline) mutateApplication(app *v1alpha1.Application, pr int) error {
	if app.Labels == nil {
		app.Labels = map[string]string{}
	}

	app.Labels[appPipelineLabel] = p.pipeline.Name
	app.Labels[pullReqIDLabel] = strconv.Itoa(pr)

	tmpl := p.pipeline.Spec.ApplicationTemplate
	if len(tmpl.Spec.Command) != 0 {
		app.Spec.Command = tmpl.Spec.Command
	}
	if len(tmpl.Spec.Args) != 0 {
		app.Spec.Args = tmpl.Spec.Args
	}
	if len(tmpl.Spec.Attributes) != 0 {
		if app.Spec.Attributes == nil {
			app.Spec.Attributes = map[string]string{}
		}
		for k, v := range tmpl.Spec.Attributes {
			app.Spec.Attributes[k] = v
		}
	}

	domain := fmt.Sprintf(
		"%s-%s-pr-%d.%s",
		p.pipeline.Namespace,
		p.pipeline.Name,
		pr,
		strings.TrimPrefix(p.pipeline.Spec.DomainBase, "*."),
	)

	app.Spec.Domains = []string{domain}

	if err := ctrl.SetControllerReference(p.pipeline, app, p.scheme); err != nil {
		return xerrors.Errorf("failed to set controller reference: %w", err)
	}

	return nil
}

func (p *GitHubPipeline) deleteObsoleteApps(ctx context.Context, prs []*github.PullRequest) error {
	apps := &v1alpha1.ApplicationList{}
	err := p.client.List(ctx, apps, client.MatchingLabels{
		appPipelineLabel: p.pipeline.Name,
	}, client.InNamespace(p.pipeline.Namespace))

	if err != nil {
		return xerrors.Errorf("failed to get the list of Application: %w", err)
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
		err := p.client.Delete(ctx, &v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: p.pipeline.Namespace,
			},
		})

		if err != nil {
			logger := p.logger.WithValues(
				"kind", "Application",
				"name", name,
				"namespace", p.pipeline.Namespace,
			)

			logger.Error(err, "failed to delete Application")

			errors = append(errors, err.Error())
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("Deleting Application failed: %s", strings.Join(errors, ","))
	}

	return nil
}

func (p *GitHubPipeline) prepareApplications(ctx context.Context, prs []*github.PullRequest) error {
	for _, pr := range prs {
		id := pr.GetNumber()

		app := &v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.getAppName(id),
				Namespace: p.pipeline.Namespace,
			},
		}

		logger := p.logger.WithValues(
			"kind", "Application",
			"name", app.Name,
			"namespace", app.Namespace,
		)

		if res, err := ctrl.CreateOrUpdate(ctx, p.client, app, func() error {
			return p.mutateApplication(app, id)
		}); err != nil {
			return xerrors.Errorf("failed to create/update Application(PR: %d): %w", id, err)
		} else {
			logger.Info("The Application operation succeeded", "op", res)
		}
	}

	if err := p.deleteObsoleteApps(ctx, prs); err != nil {
		return xerrors.Errorf("failed to clean up obsolete Applications: %w", err)
	}

	return nil
}
