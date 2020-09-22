package apppipeline

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v30/github"
	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/k8sclientutil"
	"github.com/modoki-paas/modoki-operator/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Register registers remoteSync handler to webhook handlers
func Register(c client.Client, logger logr.Logger) {
	rsh := &remoteSyncHandler{
		client: c,
		logger: logger.WithName("appPipelineHandler"),
	}

	webhook.Register("appPipeline", rsh.filter, rsh.operation)
}

type remoteSyncHandler struct {
	client client.Client
	logger logr.Logger
}

func (r *remoteSyncHandler) filter(event string) bool {
	return event == "pull_request"
}

func (r *remoteSyncHandler) refresh(ctx context.Context, logger logr.Logger, ap *v1alpha1.AppPipeline) {
	logger = logger.WithValues(
		"name", ap.Name,
		"namespace", ap.Namespace,
	)

	var err error
	for i := 0; i < 5; i++ {
		err = k8sclientutil.Patch(ctx, r.client, ap, k8sclientutil.RefreshPatch)
		if err == nil {
			return
		}

		logger.Error(err, "failed to refresh")
	}

	logger.Error(err, "failed 5 times")

	return
}

func (r *remoteSyncHandler) pullRequest(event *github.PullRequestEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	owner := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	logger := r.logger.WithValues("owner", owner, "repo", repo, "pr", event.GetNumber())

	// TODO: Should use a better approach. List() every time will cause too much load
	list := &v1alpha1.AppPipelineList{}
	if err := r.client.List(ctx, list); err != nil {
		logger.Error(err, "failed to list RemoteSync", "owner", owner, "repo", repo)
	}

	for i := range list.Items {
		item := &list.Items[i]
		gh := item.Spec.Base.GitHub

		if gh.Owner != owner ||
			gh.Repository != repo {
			continue
		}

		r.refresh(ctx, logger, item)
	}
}

func (r *remoteSyncHandler) operation(event string, payload []byte) {
	switch event {
	case "pull_request":
		event := &github.PullRequestEvent{}
		if err := json.Unmarshal(payload, event); err != nil {
			log.Printf("push payload is invalid: %+v", err)

			return
		}

		action := event.GetAction()

		if action == "opened" ||
			action == "closed" ||
			action == "reopened" {
			r.pullRequest(event)
		}
	}
}