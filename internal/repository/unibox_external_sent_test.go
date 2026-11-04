package repository

import (
	"strings"
	"testing"
)

func TestUniboxHideExternalSentSQLRespectsKillSwitch(t *testing.T) {
	t.Setenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED", "false")
	if got := uniboxHideExternalSentSQL("ue"); got != "" {
		t.Fatalf("disabled SQL = %q, want empty", got)
	}

	t.Setenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED", "true")
	got := uniboxHideExternalSentSQL("ue")
	for _, need := range []string{
		"ue.folder IS DISTINCT FROM 'sent'",
		"EXISTS",
		"tasks t",
		"BTRIM(ue.message_id",
	} {
		if !strings.Contains(got, need) {
			t.Fatalf("enabled SQL missing %q: %s", need, got)
		}
	}
}
