package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_GeneratesResetFileForMarkedStruct(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resettest\n\ngo 1.24\n")
	writeTestFile(t, root, "internal/sample/types.go", `package sample

// generate:reset
type Resettable struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *Child
}

type Child struct {
	name string
}

func (c *Child) Reset() {
	if c == nil {
		return
	}
	c.name = ""
}
`)

	if err := run(root); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	genPath := filepath.Join(root, "internal/sample/reset.gen.go")
	content, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("read generated file failed: %v", err)
	}

	got := string(content)
	mustContain(t, got, "func (r *Resettable) Reset()")
	mustContain(t, got, "if r.strP != nil {")
	mustContain(t, got, "r.s = r.s[:0]")
	mustContain(t, got, "clear(r.m)")
	mustContain(t, got, "if r.child != nil {")
}

func TestRun_GeneratesOnlyForMarkedPackages(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/resettest\n\ngo 1.24\n")

	writeTestFile(t, root, "a/types.go", `package a

// generate:reset
type A struct {
	x int
}
`)
	writeTestFile(t, root, "b/types.go", `package b

type B struct {
	x int
}
`)

	if err := run(root); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "a/reset.gen.go")); err != nil {
		t.Fatalf("expected generated file for package a: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "b/reset.gen.go")); err == nil {
		t.Fatal("did not expect generated file for package b")
	}
}

func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func mustContain(t *testing.T, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected generated code to contain %q\nactual:\n%s", want, content)
	}
}
