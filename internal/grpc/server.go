package grpcserver

import (
	"context"
	"errors"
	"strings"

	"github.com/avitamin/go-shortener/internal/auth"
	pb "github.com/avitamin/go-shortener/internal/grpc/pb"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/avitamin/go-shortener/internal/transport/validation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server is gRPC transport facade for ShortenerService.
type Server struct {
	pb.UnimplementedShortenerServiceServer
	svc *service.ShortenerService
}

// NewServer creates gRPC ShortenerService server implementation.
func NewServer(svc *service.ShortenerService) *Server {
	return &Server{svc: svc}
}

// ShortenURL creates short URL for original URL.
func (s *Server) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	orig := strings.TrimSpace(req.GetUrl())
	if orig == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	if err := validation.ValidateOriginalURL(orig); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if short, ok := s.svc.GetShort(ctx, orig); ok {
		return &pb.URLShortenResponse{Result: short}, nil
	}

	short, err := s.svc.Shorten(ctx, orig)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to shorten url")
	}

	return &pb.URLShortenResponse{Result: short}, nil
}

// ExpandURL resolves short URL id into original URL.
func (s *Server) ExpandURL(_ context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	original, err := s.svc.Resolve(id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, status.Error(codes.NotFound, "url not found")
		case errors.Is(err, repository.ErrIsDeleted):
			return nil, status.Error(codes.FailedPrecondition, "url is deleted")
		default:
			return nil, status.Error(codes.Internal, "failed to expand url")
		}
	}

	return &pb.URLExpandResponse{Result: original}, nil
}

// ListUserURLs returns URLs created by authenticated user.
func (s *Server) ListUserURLs(ctx context.Context, _ *pb.ListUserURLsRequest) (*pb.UserURLsResponse, error) {
	urls, err := s.svc.GetUserURLs(ctx)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoUserIDInContext):
			return nil, status.Error(codes.Unauthenticated, "authorization required")
		default:
			return nil, status.Error(codes.Internal, "failed to get user urls")
		}
	}

	resp := &pb.UserURLsResponse{Url: make([]*pb.URLData, 0, len(urls))}
	for _, it := range urls {
		resp.Url = append(resp.Url, &pb.URLData{
			ShortUrl:    it.Short,
			OriginalUrl: it.Original,
		})
	}

	return resp, nil
}

// UnaryAuthInterceptor injects user ID from metadata authorization token.
// ListUserURLs requires a valid token and returns Unauthenticated otherwise.
func UnaryAuthInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		authorization := ""
		if ok {
			values := md.Get("authorization")
			for _, val := range values {
				if strings.TrimSpace(val) != "" {
					authorization = val
					break
				}
			}
		}

		if strings.TrimSpace(authorization) == "" {
			if info.FullMethod == pb.ShortenerService_ListUserURLs_FullMethodName {
				return nil, status.Error(codes.Unauthenticated, "authorization metadata is required")
			}

			return handler(ctx, req)
		}

		userID, err := auth.UserIDFromAuthorization(authorization, secret)
		if err != nil {
			if info.FullMethod == pb.ShortenerService_ListUserURLs_FullMethodName {
				return nil, status.Error(codes.Unauthenticated, "invalid authorization metadata")
			}

			return handler(ctx, req)
		}

		ctx = model.NewContextWithUser(ctx, userID)
		return handler(ctx, req)
	}
}
