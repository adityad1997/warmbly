package config

import (
	"os"
	"strconv"
	"strings"
)

// MailboxExternalSendGapEnabled folds synced Sent mail (including sends from
// the provider UI) into the mailbox min-gap clock used by campaign and warmup
// scheduling, execute-time rechecks, and campaign send claims. On unless set
// to false. Gap length remains each mailbox's min_wait_time / sending-behaviour
// floor; this only widens the "last send" signal beyond Warmbly's own tasks.
func MailboxExternalSendGapEnabled() bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED")))
	return err != nil || enabled
}
