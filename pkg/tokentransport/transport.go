package tokentransport

import "net/http"

type transport struct {
	staticToken string

	Transport http.RoundTripper
}

// New returns a transport inserting a static token in the header
func New(token string) http.RoundTripper {
	return &transport{
		staticToken: token,
		Transport:   http.DefaultTransport,
	}
}

var _ http.RoundTripper = &transport{}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "token "+t.staticToken)
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	resp, err := t.Transport.RoundTrip(req)

	return resp, err
}
