package remotesync

import (
	"context"
	"encoding/json"
	"log"
	"strings"
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
		logger: logger.WithName("remoteSyncHandler"),
	}

	webhook.Register("remoteSync", rsh.filter, rsh.operation)
}

type remoteSyncHandler struct {
	client client.Client
	logger logr.Logger
}

func (r *remoteSyncHandler) filter(event string) bool {
	return event == "push" || event == "pull_request"
}

func (r *remoteSyncHandler) push(event *github.PushEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	owner := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	ref := event.GetRef()
	logger := r.logger.WithValues("owner", owner, "repo", repo, "ref", ref)

	branchPrefix := "refs/heads/"

	if !strings.HasPrefix(ref, branchPrefix) {
		return
	}
	branch := strings.TrimPrefix(ref, branchPrefix)

	// TODO: Should use a better approach. List() every time will cause too much load
	list := &v1alpha1.RemoteSyncList{}
	if err := r.client.List(ctx, list); err != nil {
		logger.Error(err, "failed to list RemoteSync", "owner", owner, "repo", repo)
	}

	for i := range list.Items {
		item := &list.Items[i]
		gh := item.Spec.Base.GitHub

		if gh.Owner == owner &&
			gh.Repository == repo &&
			gh.Branch == branch {
			if err := k8sclientutil.Patch(ctx, r.client, item, k8sclientutil.RefreshPatch); err != nil {
				logger.Error(err, "failed to refresh", "namespace", item.Namespace, "name", item.Name)
			}
		}
	}
}

func (r *remoteSyncHandler) pullRequest(event *github.PullRequestEvent) {

}

func (r *remoteSyncHandler) operation(event string, payload []byte) {
	switch event {
	case "push":
		event := &github.PushEvent{}
		if err := json.Unmarshal(payload, event); err != nil {
			log.Printf("push payload is invalid: %+v", err)

			return
		}

		r.push(event)
	case "pull_request":
		event := &github.PullRequestEvent{}
		if err := json.Unmarshal(payload, event); err != nil {
			log.Printf("push payload is invalid: %+v", err)

			return
		}

		action := event.GetAction()

		if action == "synchronize" {
			r.pullRequest(event)
		}
	}
}
