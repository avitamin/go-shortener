package repository

import (
	"context"
	"testing"

	"github.com/avitamin/go-shortener/internal/model"
)

func TestInMemoryStorage_BasicFlow(t *testing.T) {
	r := NewInMemoryStorage()
	ctx := context.Background()

	url := model.URL{Short: "abc", Original: "https://example.com", UserID: "u1"}
	if err := r.Save(ctx, url); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := r.Find("abc")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if got.Original != url.Original {
		t.Fatalf("unexpected original: %q", got.Original)
	}

	short, ok := r.GetShort(ctx, url.Original)
	if !ok || short != "abc" {
		t.Fatalf("getshort mismatch: %q, %v", short, ok)
	}

	all := r.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 item, got %d", len(all))
	}

	if err := r.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestInMemoryStorage_NotFoundAndDeleted(t *testing.T) {
	r := NewInMemoryStorage()
	ctx := context.Background()

	if _, err := r.Find("missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := r.Save(ctx, model.URL{Short: "d1", Original: "https://a", UserID: "u1"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := r.DeleteUserURLs(ctx, "u1", []string{"d1"}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := r.Find("d1"); err != ErrIsDeleted {
		t.Fatalf("expected ErrIsDeleted, got %v", err)
	}
}

func TestInMemoryStorage_SaveBatchAndContext(t *testing.T) {
	r := NewInMemoryStorage()
	ctx := context.Background()

	batch := []model.URL{
		{Short: "a", Original: "https://1", UserID: "u1"},
		{Short: "b", Original: "https://2", UserID: "u2"},
	}
	if err := r.SaveBatch(ctx, batch); err != nil {
		t.Fatalf("savebatch failed: %v", err)
	}

	if _, err := r.Find("a"); err != nil {
		t.Fatalf("find a failed: %v", err)
	}
	if _, err := r.Find("b"); err != nil {
		t.Fatalf("find b failed: %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.SaveBatch(cancelled, batch); err == nil {
		t.Fatal("expected context error")
	}
	if _, ok := r.GetShort(cancelled, "https://1"); ok {
		t.Fatal("expected no result when context is cancelled")
	}
	if err := r.DeleteUserURLs(cancelled, "u1", []string{"a"}); err == nil {
		t.Fatal("expected context error from delete")
	}
}

func TestInMemoryStorage_DeleteUserURLsOwnershipAndPing(t *testing.T) {
	r := NewInMemoryStorage()
	ctx := context.Background()

	if err := r.Save(ctx, model.URL{Short: "u1s", Original: "https://u1", UserID: "u1"}); err != nil {
		t.Fatalf("save u1 failed: %v", err)
	}
	if err := r.Save(ctx, model.URL{Short: "u2s", Original: "https://u2", UserID: "u2"}); err != nil {
		t.Fatalf("save u2 failed: %v", err)
	}

	if err := r.DeleteUserURLs(ctx, "u1", []string{"u2s", "u1s"}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := r.Find("u1s"); err != ErrIsDeleted {
		t.Fatalf("expected u1s deleted, got %v", err)
	}
	if _, err := r.Find("u2s"); err != nil {
		t.Fatalf("u2s should stay accessible, got %v", err)
	}

	if _, err := r.GetUserURLs(ctx); err != nil {
		t.Fatalf("getuserurls failed: %v", err)
	}
	if err := r.PingContext(ctx); err != ErrDBNotConfigured {
		t.Fatalf("expected ErrDBNotConfigured, got %v", err)
	}
}
