package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	pb "github.com/avitamin/go-shortener/internal/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const userCookieName = "user_id"

type cliConfig struct {
	Mode        string
	HTTPBaseURL string
	GRPCAddress string
	UseTLS      bool
	InsecureTLS bool
	Timeout     time.Duration
}

type shortenResponse struct {
	Result string `json:"result"`
}

func main() {
	cfg := parseFlags()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := run(ctx, cfg); err != nil {
		log.Printf("testsuite failed: %v", err)
		os.Exit(1)
	}

	log.Printf("testsuite %q finished successfully", cfg.Mode)
}

func parseFlags() cliConfig {
	cfg := cliConfig{}

	flag.StringVar(&cfg.Mode, "mode", "smoke", "test mode: smoke or integration")
	flag.StringVar(&cfg.HTTPBaseURL, "http-base-url", "http://localhost:8080", "HTTP base URL")
	flag.StringVar(&cfg.GRPCAddress, "grpc-address", "localhost:9090", "gRPC server address")
	flag.BoolVar(&cfg.UseTLS, "tls", false, "use TLS for gRPC and HTTPS for HTTP checks")
	flag.BoolVar(&cfg.InsecureTLS, "insecure-tls", true, "skip certificate verification for self-signed certs")
	flag.DurationVar(&cfg.Timeout, "timeout", 20*time.Second, "overall timeout")
	flag.Parse()

	return cfg
}

func run(ctx context.Context, cfg cliConfig) error {
	httpClient, err := newHTTPClient(cfg.InsecureTLS)
	if err != nil {
		return fmt.Errorf("create http client: %w", err)
	}

	grpcConn, err := newGRPCConn(cfg)
	if err != nil {
		return fmt.Errorf("connect gRPC: %w", err)
	}
	defer grpcConn.Close()

	grpcClient := pb.NewShortenerServiceClient(grpcConn)

	switch cfg.Mode {
	case "smoke":
		return runSmoke(ctx, httpClient, grpcClient, cfg.HTTPBaseURL)
	case "integration":
		return runIntegration(ctx, httpClient, grpcClient, cfg.HTTPBaseURL)
	default:
		return fmt.Errorf("unsupported mode %q", cfg.Mode)
	}
}

func runSmoke(ctx context.Context, httpClient *http.Client, grpcClient pb.ShortenerServiceClient, httpBaseURL string) error {
	original := fmt.Sprintf("https://example.com/smoke-%d", time.Now().UnixNano())
	shortURL, err := shortenHTTP(ctx, httpClient, httpBaseURL, original)
	if err != nil {
		return fmt.Errorf("smoke: http shorten failed: %w", err)
	}
	log.Printf("smoke: HTTP shorten ok: %s", shortURL)

	shortID, err := extractShortID(shortURL)
	if err != nil {
		return fmt.Errorf("smoke: parse short id: %w", err)
	}

	expanded, err := grpcClient.ExpandURL(ctx, pb.URLExpandRequest_builder{Id: shortID}.Build())
	if err != nil {
		return fmt.Errorf("smoke: grpc expand failed: %w", err)
	}

	if expanded.GetResult() != original {
		return fmt.Errorf("smoke: grpc expand mismatch: got %q want %q", expanded.GetResult(), original)
	}
	log.Printf("smoke: gRPC expand ok")

	if err := verifyHTTPRedirect(ctx, httpClient, httpBaseURL, shortID, original); err != nil {
		return fmt.Errorf("smoke: http redirect check failed: %w", err)
	}
	log.Printf("smoke: HTTP redirect ok")

	return nil
}

func runIntegration(ctx context.Context, httpClient *http.Client, grpcClient pb.ShortenerServiceClient, httpBaseURL string) error {
	originalHTTP := fmt.Sprintf("https://example.com/integration-http-%d", time.Now().UnixNano())
	shortURLHTTP, err := shortenHTTP(ctx, httpClient, httpBaseURL, originalHTTP)
	if err != nil {
		return fmt.Errorf("integration: http shorten failed: %w", err)
	}
	log.Printf("integration: HTTP shorten ok: %s", shortURLHTTP)

	token, err := extractUserToken(httpClient, httpBaseURL)
	if err != nil {
		return fmt.Errorf("integration: extract auth token from cookie: %w", err)
	}

	authCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))

	if _, err := grpcClient.ListUserURLs(ctx, pb.ListUserURLsRequest_builder{}.Build()); err == nil || status.Code(err) != codes.Unauthenticated {
		return fmt.Errorf("integration: expected Unauthenticated without metadata, got %v", err)
	}
	log.Printf("integration: gRPC unauthenticated check ok")

	if _, err := grpcClient.ListUserURLs(authCtx, pb.ListUserURLsRequest_builder{}.Build()); err != nil {
		return fmt.Errorf("integration: authorized ListUserURLs failed: %w", err)
	}
	log.Printf("integration: gRPC authorized ListUserURLs ok")

	originalGRPC := fmt.Sprintf("https://example.com/integration-grpc-%d", time.Now().UnixNano())
	grpcShort, err := grpcClient.ShortenURL(authCtx, pb.URLShortenRequest_builder{Url: originalGRPC}.Build())
	if err != nil {
		return fmt.Errorf("integration: grpc shorten failed: %w", err)
	}

	shortID, err := extractShortID(grpcShort.GetResult())
	if err != nil {
		return fmt.Errorf("integration: parse gRPC short id: %w", err)
	}

	if err := verifyHTTPRedirect(ctx, httpClient, httpBaseURL, shortID, originalGRPC); err != nil {
		return fmt.Errorf("integration: resolve gRPC-created short over HTTP failed: %w", err)
	}
	log.Printf("integration: cross-protocol resolve ok")

	return nil
}

func newHTTPClient(insecureTLS bool) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecureTLS} //nolint:gosec // self-signed certs are expected in local runtime

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
	}

	return client, nil
}

func newGRPCConn(cfg cliConfig) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{}
	if cfg.UseTLS {
		tlsCfg := &tls.Config{InsecureSkipVerify: cfg.InsecureTLS} //nolint:gosec // self-signed certs are expected in local runtime
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.NewClient(cfg.GRPCAddress, dialOpts...)
}

func shortenHTTP(ctx context.Context, client *http.Client, baseURL, original string) (string, error) {
	payload := map[string]string{"url": original}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/shorten", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var parsed shortenResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.Result) == "" {
		return "", errors.New("empty shorten result")
	}

	return parsed.Result, nil
}

func verifyHTTPRedirect(ctx context.Context, client *http.Client, baseURL, shortID, wantLocation string) error {
	resolveURL := strings.TrimRight(baseURL, "/") + "/" + shortID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolveURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != wantLocation {
		return fmt.Errorf("unexpected location %q (want %q)", got, wantLocation)
	}

	return nil
}

func extractShortID(shortURL string) (string, error) {
	u, err := url.Parse(shortURL)
	if err != nil {
		return "", err
	}

	id := strings.Trim(strings.TrimSpace(u.Path), "/")
	if id == "" {
		return "", errors.New("short id is empty")
	}

	return id, nil
}

func extractUserToken(client *http.Client, baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	for _, cookie := range client.Jar.Cookies(u) {
		if cookie.Name == userCookieName {
			if strings.TrimSpace(cookie.Value) == "" {
				return "", errors.New("user_id cookie is empty")
			}

			return cookie.Value, nil
		}
	}

	return "", errors.New("user_id cookie not found")
}
