package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/avitamin/go-shortener/internal/model"
)

func TestFileStorageRepository_SaveAndReload(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "storage.json")

	repoI, err := NewFileStorageRepository(tmpFile)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	repo := repoI.(*fileStorageRepositoy)

	ctx := context.Background()
	if err := repo.Save(ctx, model.URL{Short: "s1", Original: "https://one", UserID: "u1"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	batch := []model.URL{
		{Short: "s2", Original: "https://two", UserID: "u1"},
		{Short: "s3", Original: "https://three", UserID: "u2"},
	}
	if err := repo.SaveBatch(ctx, batch); err != nil {
		t.Fatalf("save batch failed: %v", err)
	}

	if err := repo.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	reloadedI, err := NewFileStorageRepository(tmpFile)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	reloaded := reloadedI.(*fileStorageRepositoy)
	defer reloaded.Close()

	if _, err := reloaded.Find("s1"); err != nil {
		t.Fatalf("find s1 failed: %v", err)
	}
	if _, err := reloaded.Find("s2"); err != nil {
		t.Fatalf("find s2 failed: %v", err)
	}
	if _, err := reloaded.Find("s3"); err != nil {
		t.Fatalf("find s3 failed: %v", err)
	}

	if short, ok := reloaded.GetShort(ctx, "https://two"); !ok || short != "s2" {
		t.Fatalf("unexpected getshort: %q %v", short, ok)
	}
}

func TestFileStorageRepository_LoadCorruptedLinesAndContextCancel(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "corrupted.json")
	content := "{\"short_url\":\"ok\",\"original_url\":\"https://ok\"}\nnot-json\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	repoI, err := NewFileStorageRepository(tmpFile)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	repo := repoI.(*fileStorageRepositoy)
	defer repo.Close()

	if _, err := repo.Find("ok"); err != nil {
		t.Fatalf("expected line loaded, got %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repo.SaveBatch(cancelled, []model.URL{{Short: "s4", Original: "https://four"}}); err == nil {
		t.Fatal("expected context error from SaveBatch")
	}
}

func TestFileStorageRepository_UtilityMethods(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "utility.json")
	repoI, err := NewFileStorageRepository(tmpFile)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	repo := repoI.(*fileStorageRepositoy)
	defer repo.Close()

	ctx := context.Background()
	if _, err := repo.GetUserURLs(ctx); err != nil {
		t.Fatalf("get user urls failed: %v", err)
	}
	if err := repo.DeleteUserURLs(ctx, "u1", []string{"a", "b"}); err != nil {
		t.Fatalf("delete user urls failed: %v", err)
	}
	if err := repo.PingContext(ctx); err == nil {
		t.Fatal("expected ping error")
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}
