package webhook

import (
	"log"
	"net/http"

	"github.com/google/go-github/v30/github"
)

const (
	githubEventHeader = "X-GitHub-Event"
)

// NewHandler returns handler for GitHub webhook
func NewHandler(secret string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			http.Error(rw, "not found", http.StatusNotFound)

			return
		}

		payload, err := github.ValidatePayload(req, []byte(secret))

		if err != nil {
			http.Error(rw, "token is invalid", http.StatusUnauthorized)

			return
		}

		log.Println(payload)

		call(req.Header.Get(githubEventHeader), payload)
	})
}
