package logger

import "testing"

func TestNew(t *testing.T) {
	l, err := New()
	if err != nil {
		t.Fatalf("new logger failed: %v", err)
	}
	if l == nil {
		t.Fatal("expected logger instance")
	}
	_ = l.Sync()
}
