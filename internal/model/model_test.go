package model

import (
	"context"
	"testing"
)

func TestUserFromContext(t *testing.T) {
	ctx := NewContextWithUser(context.Background(), "u1")
	u, err := UserFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "u1" {
		t.Fatalf("unexpected user id: %q", u.ID)
	}

	if _, err := UserFromContext(context.Background()); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
