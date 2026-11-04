package repository

import (
	"strings"
	"testing"
)

func TestLastUniboxSentSQLRespectsKillSwitch(t *testing.T) {
	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "false")
	if got := lastUniboxSentSQL("$1"); !strings.Contains(got, "NULL::timestamptz") {
		t.Fatalf("disabled SQL = %q, want a null stub", got)
	}
	if got := mailboxExternalSendHoldSQL(); got != "" {
		t.Fatalf("disabled claim hold = %q, want empty", got)
	}

	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "true")
	got := lastUniboxSentSQL("$1")
	for _, need := range []string{"unibox_emails", "folder = 'sent'", "email_id = $1", "LIMIT 1"} {
		if !strings.Contains(got, need) {
			t.Fatalf("enabled SQL missing %q: %s", need, got)
		}
	}
	hold := mailboxExternalSendHoldSQL()
	if !strings.Contains(hold, "unibox_emails") || !strings.Contains(hold, "NOT EXISTS") {
		t.Fatalf("enabled claim hold = %q", hold)
	}
	lateral := lastUniboxSentLateral("m.id", "at")
	if !strings.Contains(lateral, "m.id") || !strings.Contains(lateral, "AS at") {
		t.Fatalf("lateral = %q", lateral)
	}
}

func TestLastEmailTimesQueryIncludesSyncedSent(t *testing.T) {
	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "true")
	q := lastEmailTimesQuery()
	for _, need := range []string{"GREATEST(t.completed_at, ea.send_claimed_at, u.at)", "unibox_emails", "folder = 'sent'"} {
		if !strings.Contains(q, need) {
			t.Fatalf("query missing %q:\n%s", need, q)
		}
	}
}

func TestWarmupDispatchUsageQueryIncludesSyncedSent(t *testing.T) {
	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "true")
	q := warmupDispatchUsageQuery()
	for _, need := range []string{"GREATEST(", "unibox_emails", "folder = 'sent'", "FROM usage"} {
		if !strings.Contains(q, need) {
			t.Fatalf("query missing %q:\n%s", need, q)
		}
	}
}
