package emailverify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReoonMapsResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "key" {
			http.Error(w, `{"status":"error","reason":"invalid key"}`, http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/check-account-balance"):
			_, _ = w.Write([]byte(`{"api_status":"active","remaining_daily_credits":150,"remaining_instant_credits":5000,"status":"success"}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/verify"):
			if r.URL.Query().Get("mode") != "power" {
				t.Errorf("mode = %q, want power", r.URL.Query().Get("mode"))
			}
			switch r.URL.Query().Get("email") {
			case "good@x.com":
				_, _ = w.Write([]byte(`{"email":"good@x.com","status":"safe","overall_score":98,"is_safe_to_send":true,"mx_accepts_mail":true,"can_connect_smtp":true}`))
			case "catch@x.com":
				_, _ = w.Write([]byte(`{"email":"catch@x.com","status":"catch_all","is_catch_all":true,"mx_accepts_mail":true}`))
			case "role@x.com":
				_, _ = w.Write([]byte(`{"email":"role@x.com","status":"role_account","is_role_account":true,"mx_accepts_mail":true}`))
			case "gone@x.com":
				_, _ = w.Write([]byte(`{"email":"gone@x.com","status":"invalid","is_deliverable":false}`))
			case "trap@x.com":
				_, _ = w.Write([]byte(`{"email":"trap@x.com","status":"spamtrap","is_spamtrap":true}`))
			default:
				_, _ = w.Write([]byte(`{"error":"insufficient credits"}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewReoon("key", srv.URL)
	n, err := client.Account(context.Background())
	if err != nil || n == nil || *n != 5150 {
		t.Fatalf("account = %v, %v", n, err)
	}
	res, err := client.Check(context.Background(), "good@x.com")
	if err != nil || res.Status != StatusValid || res.Provider != ProviderReoon || res.Confidence != 98 || !res.HasMX {
		t.Fatalf("good = %+v, %v", res, err)
	}
	res, _ = client.Check(context.Background(), "catch@x.com")
	if res.Status != StatusRisky || res.SubStatus != SubStatusCatchAll || !res.IsCatchAll {
		t.Fatalf("catch = %+v", res)
	}
	res, _ = client.Check(context.Background(), "role@x.com")
	if res.Status != StatusRisky || res.SubStatus != SubStatusRole {
		t.Fatalf("role = %+v", res)
	}
	res, _ = client.Check(context.Background(), "gone@x.com")
	if res.Status != StatusInvalid || res.Reason != "reoon: invalid" {
		t.Fatalf("gone = %+v", res)
	}
	res, _ = client.Check(context.Background(), "trap@x.com")
	if res.Status != StatusInvalid || res.SubStatus != SubStatusSpamTrap {
		t.Fatalf("trap = %+v", res)
	}
	res, err = client.Check(context.Background(), "broke@x.com")
	if !errors.Is(err, ErrProviderCredits) || res.Status != StatusUnknown {
		t.Fatalf("no credits = %+v, %v", res, err)
	}

	bad := NewReoon("wrong", srv.URL)
	if _, err := bad.Account(context.Background()); !errors.Is(err, ErrProviderKey) {
		t.Fatalf("bad key = %v", err)
	}
}

func TestReoonTransportErrorOmitsKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	base := srv.URL
	srv.Close()

	client := NewReoon("sekrit-key", base)
	res, err := client.Check(context.Background(), "a@x.com")
	if err == nil {
		t.Fatal("expected a transport error")
	}
	if strings.Contains(err.Error(), "sekrit-key") || strings.Contains(res.Reason, "sekrit-key") {
		t.Fatalf("check error carries the key: %v / %s", err, res.Reason)
	}
	if _, err := client.Account(context.Background()); err == nil || strings.Contains(err.Error(), "sekrit-key") {
		t.Fatalf("account error carries the key or is nil: %v", err)
	}
}
