package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/warmbly/warmbly/internal/models"
)

// External Sent (provider UI) stays stored for the min-gap clock but must not
// appear in Search, Sent folder, thread reads, or Sent overview counts.

func TestLiveUniboxHidesExternalSentKeepsWarmblySent(t *testing.T) {
	t.Setenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED", "true")
	handle := liveUniboxFolderDB(t)
	f := newUniboxFolderFixture(t, handle.Pool)
	repo := NewUniboxRepository(handle)
	ctx := context.Background()
	now := time.Now().UTC()

	externalID := uuid.New()
	externalMID := "<external-" + externalID.String() + "@gmail.test>"
	if err := repo.CreateEntry(ctx, f.user, &models.EmailMessageStoreData{
		ID: externalID, EmailID: f.mailbox, Folder: models.FolderSent,
		ThreadID: "ext-" + externalID.String(), MessageID: externalMID,
		FromAddr: []string{"me@test.local"}, ToAddr: []string{"them@example.com"},
		Subject: "Sent from Gmail UI", Snippet: "outside",
		InternalDate: now, SentDate: now, CreatedAt: now, UpdatedAt: now, Seen: true,
	}); err != nil {
		t.Fatalf("create external sent: %v", err)
	}

	warmblyID := uuid.New()
	warmblyMID := "<warmbly-" + warmblyID.String() + "@campaign.test>"
	if err := repo.CreateEntry(ctx, f.user, &models.EmailMessageStoreData{
		ID: warmblyID, EmailID: f.mailbox, Folder: models.FolderSent,
		ThreadID: "wb-" + warmblyID.String(), MessageID: warmblyMID,
		FromAddr: []string{"me@test.local"}, ToAddr: []string{"lead@example.com"},
		Subject: "Campaign send", Snippet: "warmbly",
		InternalDate: now.Add(time.Minute), SentDate: now.Add(time.Minute),
		CreatedAt: now, UpdatedAt: now, Seen: true,
	}); err != nil {
		t.Fatalf("create warmbly sent: %v", err)
	}
	if _, err := handle.Pool.Exec(ctx, `INSERT INTO tasks (id, task_type, email_account_id, status, message_id, completed_at)
		VALUES ($1, 'campaign', $2, 'completed', $3, $4)`, uuid.New(), f.mailbox, warmblyMID, now); err != nil {
		t.Fatalf("stamp warmbly task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = handle.Pool.Exec(context.Background(), `DELETE FROM tasks WHERE email_account_id = $1`, f.mailbox)
	})

	sent := models.FolderSent
	res, err := repo.Search(ctx, f.org, &models.MailSearchParams{Folder: &sent, PageSize: 50})
	if err != nil {
		t.Fatalf("Search(sent): %v", err)
	}
	got := map[uuid.UUID]bool{}
	for _, row := range res.Data {
		got[row.ID] = true
	}
	if got[externalID] {
		t.Fatal("external Sent row appeared in Search(folder=sent)")
	}
	if !got[warmblyID] {
		t.Fatal("Warmbly Sent row missing from Search(folder=sent)")
	}

	thread, err := repo.GetByThread(ctx, f.org, uuid.Nil, "ext-"+externalID.String(), 20, "")
	if err != nil {
		t.Fatalf("GetByThread(external): %v", err)
	}
	if len(thread.Data) != 0 {
		t.Fatalf("GetByThread(external) = %d rows, want hidden", len(thread.Data))
	}
	thread, err = repo.GetByThread(ctx, f.org, uuid.Nil, "wb-"+warmblyID.String(), 20, "")
	if err != nil || len(thread.Data) != 1 || thread.Data[0].ID != warmblyID {
		t.Fatalf("GetByThread(warmbly)=%+v err=%v", thread, err)
	}

	ov, err := repo.Overview(ctx, f.org)
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	var sentTotal int64
	for _, folder := range ov.Folders {
		if folder.Folder == models.FolderSent {
			sentTotal = folder.Total
		}
	}
	if sentTotal != 1 {
		t.Fatalf("Sent overview total=%d, want 1 (Warmbly only)", sentTotal)
	}
}

func TestLiveUniboxHideExternalSentKillSwitchShowsOrphans(t *testing.T) {
	t.Setenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED", "false")
	handle := liveUniboxFolderDB(t)
	f := newUniboxFolderFixture(t, handle.Pool)
	repo := NewUniboxRepository(handle)
	ctx := context.Background()
	now := time.Now().UTC()

	id := uuid.New()
	if err := repo.CreateEntry(ctx, f.user, &models.EmailMessageStoreData{
		ID: id, EmailID: f.mailbox, Folder: models.FolderSent,
		ThreadID: "kill-" + id.String(), MessageID: "<kill-" + id.String() + "@gmail.test>",
		FromAddr: []string{"me@test.local"}, ToAddr: []string{"them@example.com"},
		Subject: "Visible when off", Snippet: "outside",
		InternalDate: now, SentDate: now, CreatedAt: now, UpdatedAt: now, Seen: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	sent := models.FolderSent
	res, err := repo.Search(ctx, f.org, &models.MailSearchParams{Folder: &sent, PageSize: 50})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	found := false
	for _, row := range res.Data {
		if row.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatal("kill switch off: external Sent should still appear in Search")
	}
}
