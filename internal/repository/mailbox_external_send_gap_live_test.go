package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Live coverage for the external-send gap: a synced Sent row (mail sent outside
// Warmbly) must advance the min-gap clock and hold ClaimMailboxSend.

func TestLiveExternalSentAdvancesMinGapClockAndHoldsClaim(t *testing.T) {
	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "true")
	_, pool := liveContactDB(t)
	ctx := context.Background()
	f := newWarmupUsageFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM unibox_emails WHERE email_id = $1`, f.account)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tasks WHERE email_account_id = $1`, f.account)
	})
	r := NewTaskRepository(pool).(*taskRepository)

	last, err := r.GetLastEmailTime(ctx, f.account)
	if err != nil {
		t.Fatalf("empty clock: %v", err)
	}
	if last != nil {
		t.Fatalf("empty mailbox last=%v, want nil", last)
	}

	sentAt := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	msgID := uuid.New()
	f.exec(`INSERT INTO unibox_emails (id, user_id, email_id, folder, provider_folder, message_id, internal_date, created_at, updated_at)
	        VALUES ($1, $2, $3, 'sent', 'sent', $4, $5, NOW(), NOW())`,
		msgID, f.user, f.account, "<external-"+uuid.NewString()+"@example.test>", sentAt)

	last, err = r.GetLastEmailTime(ctx, f.account)
	if err != nil || last == nil {
		t.Fatalf("clock after external sent: last=%v err=%v", last, err)
	}
	if !last.Equal(sentAt) {
		t.Fatalf("last=%v want %v", last, sentAt)
	}
	batch, err := r.GetLastEmailTimes(ctx, []uuid.UUID{f.account})
	if err != nil || !batch[f.account].Equal(sentAt) {
		t.Fatalf("batch clock=%v err=%v", batch, err)
	}

	_, _, _, usageLast, err := r.WarmupDispatchUsage(ctx, f.account, uuid.Nil,
		sentAt.Add(-time.Hour), sentAt.Add(time.Hour), sentAt.Add(-time.Hour), sentAt.Add(time.Hour))
	if err != nil || usageLast == nil || !usageLast.Equal(sentAt) {
		t.Fatalf("dispatch usage last=%v err=%v", usageLast, err)
	}

	if ok, err := r.ClaimMailboxSend(ctx, f.account, 5*time.Minute); err != nil || ok {
		t.Fatalf("claim inside external gap: ok=%v err=%v, want refused", ok, err)
	}

	f.exec(`UPDATE unibox_emails SET internal_date = NOW() - INTERVAL '20 minutes' WHERE id = $1`, msgID)
	if ok, err := r.ClaimMailboxSend(ctx, f.account, 5*time.Minute); err != nil || !ok {
		t.Fatalf("claim after external gap: ok=%v err=%v, want taken", ok, err)
	}
}

func TestLiveExternalSendGapKillSwitchIgnoresSyncedSent(t *testing.T) {
	t.Setenv("MAILBOX_EXTERNAL_SEND_GAP_ENABLED", "false")
	_, pool := liveContactDB(t)
	ctx := context.Background()
	f := newWarmupUsageFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM unibox_emails WHERE email_id = $1`, f.account)
	})
	r := NewTaskRepository(pool).(*taskRepository)

	f.exec(`INSERT INTO unibox_emails (id, user_id, email_id, folder, provider_folder, message_id, internal_date, created_at, updated_at)
	        VALUES ($1, $2, $3, 'sent', 'sent', $4, NOW() - INTERVAL '30 seconds', NOW(), NOW())`,
		uuid.New(), f.user, f.account, "<external-off-"+uuid.NewString()+"@example.test>")

	last, err := r.GetLastEmailTime(ctx, f.account)
	if err != nil {
		t.Fatalf("clock: %v", err)
	}
	if last != nil {
		t.Fatalf("kill switch on: last=%v, want nil (synced Sent ignored)", last)
	}
	if ok, err := r.ClaimMailboxSend(ctx, f.account, 5*time.Minute); err != nil || !ok {
		t.Fatalf("claim with kill switch: ok=%v err=%v, want taken", ok, err)
	}
}
