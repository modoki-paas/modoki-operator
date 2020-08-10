package handler

import (
	"context"
	"net"
	"runtime/debug"

	"github.com/go-logr/logr"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Server struct {
	logger logr.Logger
	server *grpc.Server
}

type RegisterService interface {
	Register(s *Server)
}

func NewServer(logger logr.Logger) *Server {
	s := &Server{
		logger: logger,
	}

	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(func(p interface{}) error {
			s.logger.Error(nil, "handler paniced", "param", p, "trace", string(debug.Stack()))

			return grpc.Errorf(codes.Internal, "internal error")
		}),
	}

	s.server = grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_recovery.StreamServerInterceptor(opts...),
		),
	)

	return s
}

func (s *Server) Start(ctx context.Context, address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return xerrors.Errorf("failed to open a new tcp listener: %w", err)
	}

	end := make(chan struct{}, 1)
	defer close(end)
	go func() {
		select {
		case <-ctx.Done():
		case <-end:
		}
		s.server.GracefulStop()
	}()

	if err := s.server.Serve(listener); err != nil {
		return err
	}

	return ctx.Err()
}
