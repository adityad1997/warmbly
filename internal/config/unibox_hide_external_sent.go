package config

import (
	"os"
	"strconv"
	"strings"
)

// UniboxHideExternalSentEnabled hides Sent copies that were not sent by
// Warmbly (no matching tasks.message_id) from unibox search, thread reads,
// and overview counts. Synced Sent rows stay in unibox_emails so the mailbox
// min-gap clock can still see them. On unless set to false.
func UniboxHideExternalSentEnabled() bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED")))
	return err != nil || enabled
}
