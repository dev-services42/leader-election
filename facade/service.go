package facade

import (
	"context"
	"github.com/dev-services42/leader-election/domain"
	service "github.com/dev-services42/leader-election/gen/contracts"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/reflection"
	"net"
	"sync"
)

type Service struct {
	logger   *zap.Logger
	stopWg   *sync.WaitGroup
	grpcAddr string
	domain   *domain.Service
}

func New(logger *zap.Logger, grpcAddr string, srvDomain *domain.Service) (*Service, error) {
	return &Service{
		logger:   logger,
		stopWg:   new(sync.WaitGroup),
		grpcAddr: grpcAddr,
		domain:   srvDomain,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		return errors.Wrap(err, "cannot run listener")
	}

	grpcServer, err := s.buildGrpcSerer()
	if err != nil {
		return errors.Wrap(err, "cannot build grpc server")
	}

	service.RegisterLeaderElectionServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	s.stopWg.Add(1)
	go func() {
		defer s.stopWg.Done()

		<-ctx.Done()
		s.logger.Info("stream grpc context done")
		grpcServer.Stop()
		s.logger.Info("stream grpc server stopped")
	}()

	s.stopWg.Add(1)
	go func() {
		defer s.stopWg.Done()

		s.logger.Info("started grpc server")
		if errServe := grpcServer.Serve(listener); errServe != nil {
			s.logger.Error("server has been stopped", zap.Error(err))
		}
	}()

	return nil
}

func (s *Service) Wait() {
	s.stopWg.Wait()
}
