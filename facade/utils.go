package facade

import (
	"context"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"runtime/debug"
	"time"
)

func getLoggingUnaryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		id := uuid.New().String()
		start := time.Now()
		logger = logger.With(
			zap.String("request-id", id),
			zap.String("method", info.FullMethod),
			zap.Any("request", req),
		)

		resp, err := handler(ctx, req)
		since := time.Since(start)
		logger = logger.With(zap.Duration("duration", since))
		if err != nil {
			logger.Error("cannot handle request", zap.Error(err))
			return nil, err
		}

		logger.Info("handle request", zap.Any("response", resp))
		return resp, nil
	}
}

func getLoggingStreamInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(s interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		id := uuid.New().String()
		logger := logger.With(
			zap.String("request-id", id),
			zap.String("method", info.FullMethod),
			// TODO: оборачиваться воркуг стрима и логировать запросы и ответы
			zap.String("request", "<cannot extract body from stream>"),
			zap.String("response", "<cannot extract body from stream>"),
		)

		logger.Info("handling stream request")
		if err := handler(s, stream); err != nil {
			logger.Error("cannot handle request", zap.Error(err))
			return status.Errorf(status.Code(err), "%v", err)
		}

		logger.Info("stream closed")

		return nil
	}
}

func (s *Service) buildGrpcSerer() (*grpc.Server, error) {
	grpcPanicRecoveryFunc := func(panicValue interface{}) (err error) {
		s.logger.Error("panic during handling gRPC request",
			zap.Reflect("panic_value", panicValue),
			zap.String("stack_trace", string(debug.Stack())),
		)
		return status.Errorf(codes.Unknown, "Internal Server Error")
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		getLoggingUnaryInterceptor(s.logger),
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		getLoggingStreamInterceptor(s.logger),
		grpc_recovery.StreamServerInterceptor(
			grpc_recovery.WithRecoveryHandler(grpcPanicRecoveryFunc),
		),
	}

	const KB = 1024
	options := []grpc.ServerOption{
		grpc.WriteBufferSize(1 * KB),
		grpc.ReadBufferSize(1 * KB),
		grpc.InitialWindowSize(1 * KB),
		grpc.InitialConnWindowSize(1 * KB),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 30 * time.Second,
		}),
		grpc.MaxRecvMsgSize(1 * KB),
		grpc.MaxHeaderListSize(2 * KB),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
	}

	server := grpc.NewServer(options...)

	return server, nil
}
