package pollclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type mockNetError struct {
	err string
}

func (m mockNetError) Error() string   { return m.err }
func (m mockNetError) Timeout() bool   { return true }
func (m mockNetError) Temporary() bool { return true }

func TestIsUpstreamUnavailable_NetworkErrors(t *testing.T) {
	tests := []error{
		&url.Error{Err: mockNetError{err: "timeout"}},
		&url.Error{Err: &net.DNSError{}},
		&url.Error{Err: &net.OpError{}},
	}
	for i, err := range tests {
		if !isUpstreamUnavailable(err) {
			t.Fatalf("case %d: expected upstream unavailable", i)
		}
	}
}

func TestIsUpstreamUnavailable_NonNetwork(t *testing.T) {
	err := errors.New("not network")
	if isUpstreamUnavailable(err) {
		t.Fatalf("expected non-network error to be false")
	}
}

func TestPollOnceNon2xxIsNotUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		PollPath:   "/",
		Interval:   time.Second,
		HTTPClient: server.Client(),
	})
	if err := client.pollOnce(context.Background()); err == nil {
		t.Fatalf("expected error from non-2xx poll")
	} else if isUpstreamUnavailable(err) {
		t.Fatalf("non-2xx should not be classified as upstream unavailable")
	}
}


