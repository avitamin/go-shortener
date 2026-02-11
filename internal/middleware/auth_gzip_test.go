package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avitamin/go-shortener/internal/model"
	"go.uber.org/zap"
)

func TestAuth_NoCookieCreatesSignedCookie(t *testing.T) {
	secret := "test-secret"
	var userID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := r.Context().Value(model.ContextUserID).(string); ok {
			userID = id
		}
		w.WriteHeader(http.StatusOK)
	})

	h := Auth(secret, zap.NewNop())(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected signed cookie")
	}
	if userID == "" {
		t.Fatal("expected user id in context")
	}
}

func TestAuth_ValidCookiePassesUser(t *testing.T) {
	secret := "test-secret"
	expectedID := "user-123"
	sig := computeHMAC(expectedID, secret)

	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(model.ContextUserID).(string)
		gotID = id
		w.WriteHeader(http.StatusOK)
	})

	h := Auth(secret, zap.NewNop())(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: expectedID + "|" + sig})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if gotID != expectedID {
		t.Fatalf("unexpected user id: %q", gotID)
	}
}

func TestAuth_BadCookieUnauthorized(t *testing.T) {
	h := Auth("secret", zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "broken"})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidSignatureIssuesNewCookie(t *testing.T) {
	h := Auth("secret", zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "user-1|bad-signature"})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("expected new cookie")
	}
}

func TestVerifySignature(t *testing.T) {
	sig := computeHMAC("u1", "s1")
	if !verifySignature("u1", sig, "s1") {
		t.Fatal("expected valid signature")
	}
	if verifySignature("u1", "wrong", "s1") {
		t.Fatal("expected invalid signature")
	}
}

func TestGzipRequest(t *testing.T) {
	logger := zap.NewNop()
	h := GzipRequest(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body failed: %v", err)
		}
		_, _ = w.Write(body)
	}))

	plainReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("plain"))
	plainW := httptest.NewRecorder()
	h.ServeHTTP(plainW, plainReq)
	if plainW.Body.String() != "plain" {
		t.Fatalf("unexpected plain result: %q", plainW.Body.String())
	}

	var gzBody bytes.Buffer
	gz := gzip.NewWriter(&gzBody)
	_, _ = gz.Write([]byte("compressed"))
	_ = gz.Close()

	gzReq := httptest.NewRequest(http.MethodPost, "/", &gzBody)
	gzReq.Header.Set("Content-Encoding", "gzip")
	gzW := httptest.NewRecorder()
	h.ServeHTTP(gzW, gzReq)
	if gzW.Body.String() != "compressed" {
		t.Fatalf("unexpected gzip result: %q", gzW.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not-gzip"))
	badReq.Header.Set("Content-Encoding", "gzip")
	badW := httptest.NewRecorder()
	h.ServeHTTP(badW, badReq)
	if badW.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", badW.Code)
	}
}

func TestGzipResponse(t *testing.T) {
	h := GzipResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	plainReq := httptest.NewRequest(http.MethodGet, "/", nil)
	plainW := httptest.NewRecorder()
	h.ServeHTTP(plainW, plainReq)
	if plainW.Header().Get("Content-Encoding") != "" {
		t.Fatal("did not expect gzip header")
	}
	if plainW.Body.String() != "hello" {
		t.Fatalf("unexpected plain body: %q", plainW.Body.String())
	}

	gzReq := httptest.NewRequest(http.MethodGet, "/", nil)
	gzReq.Header.Set("Accept-Encoding", "gzip")
	gzW := httptest.NewRecorder()
	h.ServeHTTP(gzW, gzReq)
	if gzW.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip header")
	}
	zr, err := gzip.NewReader(bytes.NewReader(gzW.Body.Bytes()))
	if err != nil {
		t.Fatalf("new gzip reader failed: %v", err)
	}
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read decoded failed: %v", err)
	}
	if string(decoded) != "hello" {
		t.Fatalf("unexpected decoded body: %q", string(decoded))
	}
}
