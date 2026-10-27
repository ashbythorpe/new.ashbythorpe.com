package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestCommentThreads(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	author := fixtureUser(t, "author@example.com", true)
	other := fixtureUser(t, "other@example.com", true)
	root, err := CreateComment(ctx, "post", author, "Root", nil)
	if err != nil {
		t.Fatal(err)
	}
	if root.OriginalReplyTo != nil {
		t.Fatal("root has parent")
	}
	if _, err := time.Parse(time.RFC3339, root.CreatedAt); err != nil {
		t.Fatal(err)
	}
	reply, err := CreateComment(ctx, "post", other, "Reply", &root.ID)
	if err != nil {
		t.Fatal(err)
	}
	nested, err := CreateComment(ctx, "post", author, "Nested", &reply.ID)
	if err != nil {
		t.Fatal(err)
	}
	if nested.OriginalReplyTo == nil || *nested.OriginalReplyTo != root.ID {
		t.Fatal("nested reply lost root")
	}
	comments, err := GetComments(ctx, "post", 1)
	if err != nil || len(comments) != 1 {
		t.Fatalf("comments=%+v, %v", comments, err)
	}
	if comments[0].NumReplies != 2 || comments[0].Author.ID != author || comments[0].Content != "Root" {
		t.Fatalf("comment=%+v", comments[0])
	}
	replies, err := GetReplies(ctx, "post", root.ID)
	if err != nil || len(replies) != 2 {
		t.Fatalf("replies=%+v, %v", replies, err)
	}
	found := false
	for _, r := range replies {
		if r.ID == nested.ID {
			found = true
			if r.ReplyTo == nil || r.ReplyTo.ID != reply.ID || r.ReplyTo.Name != "other@example.com" {
				t.Fatalf("nested=%+v", r)
			}
		}
	}
	if !found {
		t.Fatal("nested reply missing")
	}
	n, err := CountComments(ctx, "post")
	if err != nil || n != 1 {
		t.Fatalf("count=%d, %v", n, err)
	}
	if _, err := CreateComment(ctx, "different-post", author, "Cross-post", &root.ID); err == nil {
		t.Fatal("cross-post reply accepted")
	}
	_, err = CreateComment(ctx, "post", author, "Missing parent", new(999))
	wantError(t, err, sql.ErrNoRows)
	empty, err := GetReplies(ctx, "different-post", root.ID)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty replies=%+v, %v", empty, err)
	}
}

func TestCommentOwnershipAndDeletion(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	owner := fixtureUser(t, "owner@example.com", true)
	other := fixtureUser(t, "other@example.com", true)
	root, err := CreateComment(ctx, "post", owner, "Original", nil)
	if err != nil {
		t.Fatal(err)
	}
	reply, err := CreateComment(ctx, "post", other, "Reply", &root.ID)
	if err != nil {
		t.Fatal(err)
	}
	nested, err := CreateComment(ctx, "post", owner, "Nested", &reply.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = EditComment(ctx, root.ID, other, "Unauthorized")
	wantError(t, err, sql.ErrNoRows)
	_, err = DeleteComment(ctx, root.ID, other)
	wantError(t, err, sql.ErrNoRows)
	comments, err := GetComments(ctx, "post", 1)
	if err != nil || len(comments) != 1 || comments[0].Content != "Original" {
		t.Fatalf("unauthorized edit changed comment: %+v, %v", comments, err)
	}
	parent, err := EditComment(ctx, nested.ID, owner, "Edited")
	if err != nil || parent == nil || *parent != root.ID {
		t.Fatalf("edit parent=%v, %v", parent, err)
	}
	parent, err = DeleteComment(ctx, reply.ID, other)
	if err != nil || parent == nil || *parent != root.ID {
		t.Fatalf("delete parent=%v, %v", parent, err)
	}
	replies, err := GetReplies(ctx, "post", root.ID)
	if err != nil || len(replies) != 1 || replies[0].ReplyTo != nil || replies[0].Content != "Edited" {
		t.Fatalf("remaining replies=%+v, %v", replies, err)
	}
	if _, err := DeleteComment(ctx, root.ID, owner); err != nil {
		t.Fatal(err)
	}
	if countRows(t, "comments") != 0 {
		t.Fatal("thread not cascaded")
	}
}

func TestCommentPaginationAndPostIsolation(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	author := fixtureUser(t, "author@example.com", true)
	for i := range 12 {
		comment, err := CreateComment(ctx, "post", author, fmt.Sprint(i), nil)
		if err != nil {
			t.Fatal(err)
		}
		execSQL(t, "UPDATE comments SET created_at=? WHERE id=?", time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC).Format("2006-01-02 15:04:05"), comment.ID)
	}
	if _, err := CreateComment(ctx, "other", author, "Other post", nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		page, n int
		first   string
	}{{1, 10, "11"}, {2, 2, "1"}, {3, 0, ""}} {
		comments, err := GetComments(ctx, "post", tc.page)
		if err != nil || comments == nil || len(comments) != tc.n {
			t.Fatalf("page %d=%+v, %v", tc.page, comments, err)
		}
		if tc.n > 0 && comments[0].Content != tc.first {
			t.Fatalf("page %d order=%+v", tc.page, comments)
		}
	}
	n, err := CountComments(ctx, "post")
	if err != nil || n != 12 {
		t.Fatalf("count=%d, %v", n, err)
	}
}

func TestCleanupExpiredRecords(t *testing.T) {
	testDB(t)
	id := fixtureUser(t, "user@example.com", true)
	past := time.Now().Add(-time.Hour).Unix()
	future := time.Now().Add(time.Hour).Unix()
	for _, table := range []string{"sessions", "password_reset_tokens"} {
		column := "token"
		if table == "sessions" {
			column = "id"
		}
		for i, expiry := range []int64{past, future} {
			execSQL(t, "INSERT INTO "+table+" ("+column+",user_id,expiry) VALUES (?,?,?)", []byte{byte(i)}, id[:], expiry)
		}
	}
	other := fixtureUser(t, "other@example.com", true)
	execSQL(t, "INSERT INTO verification_tokens (token,user_id,expiry) VALUES (?,?,?),(?,?,?)", []byte{0}, id[:], past, []byte{1}, other[:], future)
	if err := NewDBCleanupService(DB).cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"sessions", "password_reset_tokens", "verification_tokens"} {
		if countRows(t, table) != 1 {
			t.Fatalf("cleanup %s", table)
		}
		var expiry int64
		if err := DB.QueryRow("SELECT expiry FROM " + table).Scan(&expiry); err != nil || expiry != future {
			t.Fatalf("retained expiry=%d, %v", expiry, err)
		}
	}
}
