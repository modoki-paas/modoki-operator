package handler

import (
	"context"
	"time"

	"github.com/modoki-paas/modoki-operator/proto/webhook"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type WebhookServer struct {
	ch chan<- event.GenericEvent
}

func NewWebhookServer(ch chan<- event.GenericEvent) *WebhookServer {
	return &WebhookServer{
		ch: ch,
	}
}

var _ webhook.WebhookServer = (*WebhookServer)(nil)
var _ RegisterService = (*WebhookServer)(nil)

func (ws *WebhookServer) Register(s *Server) {
	webhook.RegisterWebhookServer(s.server, ws)
}

// Enqueue handles requests to reconcile a resource
func (ws *WebhookServer) Enqueue(ctx context.Context, req *webhook.EnqueueRequest) (*webhook.EnqueueResponse, error) {
	ge := event.GenericEvent{
		Meta: &metav1.ObjectMeta{
			Name:      req.Metadata.Name,
			Namespace: req.Metadata.Namespace,
		},
	}

	timer := time.NewTimer(30 * time.Second)
	select {
	case ws.ch <- ge:
	case <-timer.C:
		return nil, grpc.Errorf(codes.ResourceExhausted, "enqueueing timed out")
	}

	return &webhook.EnqueueResponse{}, nil
}
