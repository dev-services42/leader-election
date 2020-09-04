package facade

import (
	service "github.com/dev-services42/leader-election/gen/contracts"
	"github.com/pkg/errors"
)

func (s *Service) SubscribeOnLeader(req *service.SubscribeOnLeaderRequest, srv service.LeaderElectionService_SubscribeOnLeaderServer) error {
	ctx := srv.Context()
	ch := s.domain.SubscribeOnLeader(ctx)

	for {
		select {
		case <-ctx.Done():
		case isMaster, ok := <-ch:
			if !ok {
				return nil
			}

			err := srv.Send(&service.SubscribeOnLeaderResponse{
				IsLeader: isMaster,
			})
			if err != nil {
				return errors.Wrap(err, "cannot send to response channel")
			}
		}
	}
}
