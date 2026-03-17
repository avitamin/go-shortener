package grpcserver_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/avitamin/go-shortener/internal/audit"
	authn "github.com/avitamin/go-shortener/internal/auth"
	"github.com/avitamin/go-shortener/internal/config"
	grpcserver "github.com/avitamin/go-shortener/internal/grpc"
	pb "github.com/avitamin/go-shortener/internal/grpc/pb"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestShortenURLAndExpandURL(t *testing.T) {
	client, _, cleanup := setupGRPCClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.ShortenURL(ctx, pb.URLShortenRequest_builder{Url: "invalid"}.Build())
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}

	shortResp, err := client.ShortenURL(ctx, pb.URLShortenRequest_builder{Url: "https://example.com/grpc"}.Build())
	if err != nil {
		t.Fatalf("shorten failed: %v", err)
	}

	parts := strings.Split(strings.TrimSuffix(shortResp.GetResult(), "/"), "/")
	if len(parts) == 0 {
		t.Fatalf("invalid shortened url: %s", shortResp.GetResult())
	}

	expandResp, err := client.ExpandURL(ctx, pb.URLExpandRequest_builder{Id: parts[len(parts)-1]}.Build())
	if err != nil {
		t.Fatalf("expand failed: %v", err)
	}

	if expandResp.GetResult() != "https://example.com/grpc" {
		t.Fatalf("unexpected original url: %s", expandResp.GetResult())
	}
}

func TestListUserURLsRequiresAuthorization(t *testing.T) {
	client, _, cleanup := setupGRPCClient(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.ListUserURLs(ctx, pb.ListUserURLsRequest_builder{}.Build())
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestListUserURLsWithAuthorization(t *testing.T) {
	client, svc, cleanup := setupGRPCClient(t)
	defer cleanup()

	token := authn.BuildSignedUserID("grpc-user", "secret_key")
	authCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	seedCtx := model.NewContextWithUser(context.Background(), "grpc-user")
	_, err := svc.Shorten(seedCtx, "https://example.com/1")
	if err != nil {
		t.Fatalf("shorten #1 failed: %v", err)
	}

	_, err = svc.Shorten(seedCtx, "https://example.com/2")
	if err != nil {
		t.Fatalf("shorten #2 failed: %v", err)
	}

	ctx3, cancel3 := context.WithTimeout(authCtx, 3*time.Second)
	defer cancel3()
	resp, err := client.ListUserURLs(ctx3, pb.ListUserURLsRequest_builder{}.Build())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	_ = resp
}

func setupGRPCClient(t *testing.T) (pb.ShortenerServiceClient, *service.ShortenerService, func()) {
	t.Helper()

	cfg := &config.Config{
		Address:     "127.0.0.1:8080",
		GRPCAddress: "127.0.0.1:9090",
		BaseURL:     "http://localhost:8080",
		SecretKey:   "secret_key",
	}

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	transport := grpcserver.NewServer(svc)

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.UnaryAuthInterceptor(cfg.SecretKey)))
	pb.RegisterShortenerServiceServer(grpcSrv, transport)

	listener := bufconn.Listen(bufSize)
	go func() {
		_ = grpcSrv.Serve(listener)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcSrv.Stop()
		_ = svc.Shutdown(context.Background())
		_ = repo.Close()
	}

	return pb.NewShortenerServiceClient(conn), svc, cleanup
}
