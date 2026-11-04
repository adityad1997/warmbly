package repository

import "github.com/warmbly/warmbly/internal/config"

// uniboxHideExternalSentSQL is AND-ed onto a unibox_emails alias so Sent rows
// with no matching Warmbly task (mail sent from the provider UI, or otherwise
// outside Warmbly) stay out of list, thread, and overview queries. Non-sent
// folders are unchanged. Empty when the feature is off.
//
// Storage is intentionally untouched: synced Sent remains in unibox_emails for
// the mailbox min-gap clock. Matching uses both bracketed and bare Message-ID
// forms, the same way GetTaskByMessageID does.
func uniboxHideExternalSentSQL(alias string) string {
	if !config.UniboxHideExternalSentEnabled() {
		return ""
	}
	return ` AND (` + alias + `.folder IS DISTINCT FROM 'sent' OR ` + warmblyOriginSentExistsSQL(alias) + `)`
}

func warmblyOriginSentExistsSQL(alias string) string {
	mid := alias + `.message_id`
	return `EXISTS (
		SELECT 1 FROM tasks t
		WHERE t.message_id <> ''
		  AND (
		    t.message_id = BTRIM(` + mid + `, '<> ')
		    OR t.message_id = '<' || BTRIM(` + mid + `, '<> ') || '>'
		  )
	)`
}
