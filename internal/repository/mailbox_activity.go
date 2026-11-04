package repository

import "github.com/warmbly/warmbly/internal/config"

// lastUniboxSentSQL is a scalar subquery for the latest synced Sent row on a
// mailbox (unibox_emails.email_id = accountExpr). The min-gap clock GREATEST()s
// this with Warmbly task completions and send_claimed_at so a send from Gmail,
// Outlook, or another client that has already been synced holds the same gap
// as a Warmbly send.
//
// Best-effort only: an external send that has not synced yet is invisible here.
// Warmbly's own send still advances the task / claim clocks immediately, so a
// later sync of that same Sent copy cannot shrink the gap.
func lastUniboxSentSQL(accountExpr string) string {
	if !config.MailboxExternalSendGapEnabled() {
		return `(SELECT NULL::timestamptz)`
	}
	return `(SELECT ue.internal_date FROM unibox_emails ue` +
		` WHERE ue.email_id = ` + accountExpr +
		` AND ue.folder = 'sent'` +
		` ORDER BY ue.internal_date DESC NULLS LAST, ue.id DESC LIMIT 1)`
}

// lastUniboxSentLateral is the LEFT JOIN LATERAL form used by the batch
// min-gap clock. alias is the selected column name (e.g. "at").
func lastUniboxSentLateral(accountExpr, alias string) string {
	if !config.MailboxExternalSendGapEnabled() {
		return `(SELECT NULL::timestamptz AS ` + alias + `) u ON true`
	}
	return `(SELECT ue.internal_date AS ` + alias + ` FROM unibox_emails ue` +
		` WHERE ue.email_id = ` + accountExpr +
		` AND ue.folder = 'sent'` +
		` ORDER BY ue.internal_date DESC NULLS LAST, ue.id DESC LIMIT 1) u ON true`
}

// mailboxExternalSendHoldSQL is AND-ed into ClaimMailboxSend so a campaign
// cannot take the mailbox while a recently synced Sent row still sits inside
// the gap window. Empty when the feature is off.
func mailboxExternalSendHoldSQL() string {
	if !config.MailboxExternalSendGapEnabled() {
		return ""
	}
	return `
		  AND NOT EXISTS (
		    SELECT 1 FROM unibox_emails ue
		    WHERE ue.email_id = $1
		      AND ue.folder = 'sent'
		      AND ue.internal_date > NOW() - make_interval(secs => $2))`
}
