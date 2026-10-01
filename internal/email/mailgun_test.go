package email

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"orioncaps-api/internal/config"
)

func newTestMailgun(url string, retries int) *Mailgun {
	m := NewMailgun(config.Config{
		MailgunAPIKey:  "key-test",
		MailgunDomain:  "mg.test.com",
		MailgunBaseURL: url,
		MailgunTimeout: 2 * time.Second,
		MailgunRetries: retries,
		BrandName:      "Orion Caps",
	})
	m.backoff = 0
	return m
}

func msg() Message {
	return Message{To: []string{"a@x.com", "b@x.com"}, Subject: "s", HTML: "<p>h</p>", Text: "t",
		ReplyTo: "c@x.com", Tags: []string{"t1"}}
}

func TestSendOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()
		if r.URL.Path != "/mg.test.com/messages" || user != "api" || pass != "key-test" {
			t.Errorf("request inesperado: %s %s:%s", r.URL.Path, user, pass)
		}
		_ = r.ParseForm()
		if got := r.PostForm["to"]; len(got) != 2 {
			t.Errorf("to: %v", got)
		}
		if r.PostForm.Get("h:Reply-To") != "c@x.com" || r.PostForm.Get("o:tag") != "t1" {
			t.Errorf("form: %v", r.PostForm)
		}
		if r.PostForm.Get("from") != "Orion Caps <no-reply@mg.test.com>" {
			t.Errorf("from: %s", r.PostForm.Get("from"))
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	if err := newTestMailgun(srv.URL, 1).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
}

func TestRetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	if err := newTestMailgun(srv.URL, 1).Send(context.Background(), msg()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(401)
	}))
	defer srv.Close()

	err := newTestMailgun(srv.URL, 3).Send(context.Background(), msg())
	if !errors.Is(err, ErrDelivery) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // puerto cerrado

	err := newTestMailgun(url, 1).Send(context.Background(), msg())
	if !errors.Is(err, ErrDelivery) {
		t.Fatalf("err=%v", err)
	}
}

func TestContextCancelledStopsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := newTestMailgun(srv.URL, 5).Send(ctx, msg())
	if !errors.Is(err, ErrDelivery) {
		t.Fatalf("err=%v", err)
	}
}

func TestNotConfigured(t *testing.T) {
	err := NewMailgun(config.Config{}).Send(context.Background(), msg())
	if !errors.Is(err, ErrDelivery) {
		t.Fatalf("err=%v", err)
	}
}
