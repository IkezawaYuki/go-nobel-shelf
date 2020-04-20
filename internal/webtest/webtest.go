package webtest

import (
	"net/http"
	"testing"
	"time"
)

type W struct {
	t      *testing.T
	host   string
	Client *http.Client
}

func New(t *testing.T, host string) *W {
	return &W{
		t:      t,
		host:   host,
		Client: http.DefaultClient,
	}
}

func (w *W) WaitForNet() {
	const retryDelay = 100 * time.Millisecond
	deadline := time.Now().Add(30 * time.Second)

}
