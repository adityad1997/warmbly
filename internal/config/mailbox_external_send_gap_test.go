package config

import "testing"

func TestMailboxExternalSendGapEnabled(t *testing.T) {
	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "")
	if !MailboxExternalSendGapEnabled() {
		t.Fatal("unset must default on")
	}
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"nope", true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", tc.raw)
			if got := MailboxExternalSendGapEnabled(); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
