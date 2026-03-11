package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSelfSignedCertificate(t *testing.T) {
	cert, err := generateSelfSignedCertificate("127.0.0.1:8443")
	require.NoError(t, err)
	require.NotEmpty(t, cert.Certificate)

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)

	assert.Equal(t, "go-shortener", parsed.Subject.CommonName)
	assert.Contains(t, parsed.DNSNames, "localhost")
	require.True(t, containsIP(parsed.IPAddresses, net.ParseIP("127.0.0.1")))
	assert.WithinDuration(t, time.Now(), parsed.NotBefore.Add(time.Hour), 5*time.Second)
	assert.WithinDuration(t, time.Now().Add(365*24*time.Hour), parsed.NotAfter, 5*time.Second)
}

func TestLaunchServerHTTP(t *testing.T) {
	addr := freeAddr(t)
	server := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok-http"))
		}),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- launchServer(server, false)
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	resp := waitForResponse(t, client, "http://"+addr)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok-http", string(body))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, server.Shutdown(shutdownCtx))

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

func TestLaunchServerHTTPS(t *testing.T) {
	addr := freeAddr(t)
	server := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok-https"))
		}),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- launchServer(server, true)
	}()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // проверка self-signed сертификата в тесте
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: transport}
	t.Cleanup(client.CloseIdleConnections)

	resp := waitForResponse(t, client, "https://"+addr)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok-https", string(body))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, server.Shutdown(shutdownCtx))

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	return listener.Addr().String()
}

func waitForResponse(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			return resp
		}

		lastErr = err
		time.Sleep(30 * time.Millisecond)
	}

	require.NoError(t, lastErr)
	return nil
}

func containsIP(ips []net.IP, want net.IP) bool {
	for _, ip := range ips {
		if ip.Equal(want) {
			return true
		}
	}

	return false
}
