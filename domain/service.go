package domain

import (
	"context"
	election "github.com/dev-services42/leader-election-lib/leader-election"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"sync"
	"time"
)

type subscription struct {
	Channel chan bool
	Cancel  context.CancelFunc
}

type Service struct {
	stopWg             *sync.WaitGroup
	logger             *zap.Logger
	leaderElection     *election.Service
	subscriptionsMutex *sync.RWMutex
	subscriptions      []subscription

	isMaster          *atomic.Bool
	slowClientReadTTL time.Duration
}

func New(logger *zap.Logger, leaderElection *election.Service, slowClientReadTTL time.Duration) (*Service, error) {
	return &Service{
		logger:             logger,
		stopWg:             new(sync.WaitGroup),
		leaderElection:     leaderElection,
		isMaster:           atomic.NewBool(false),
		slowClientReadTTL:  slowClientReadTTL,
		subscriptionsMutex: new(sync.RWMutex),
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	s.stopWg.Add(1)
	go func() {
		defer s.stopWg.Done()

		leaderCh := s.leaderElection.RunLeaderElection(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case isMaster, ok := <-leaderCh:
				if !ok {
					return
				}

				old := s.isMaster.Swap(isMaster)
				s.subscriptionsMutex.RLock()
				for _, ch := range s.subscriptions {
					select {
					case <-ctx.Done():
					case <-time.After(s.slowClientReadTTL):
						ch.Cancel()
					case ch.Channel <- isMaster:
					}
				}
				s.subscriptionsMutex.RUnlock()

				if old != isMaster {
					s.logger.Info("master state changed", zap.Bool("newState", isMaster))
				}
			}
		}
	}()

	return nil
}

func (s *Service) SubscribeOnLeader(ctx context.Context) <-chan bool {
	ch := make(chan bool)

	ctx2, cancel := context.WithCancel(ctx)
	s.stopWg.Add(1)
	go func() {
		defer s.stopWg.Done()
		defer close(ch)

		select {
		case <-ctx2.Done():
			return
		case ch <- s.isMaster.Load():
		}

		s.subscriptionsMutex.Lock()
		s.subscriptions = append(s.subscriptions, subscription{
			Channel: ch,
			Cancel:  cancel,
		})
		s.subscriptionsMutex.Unlock()

		<-ctx2.Done()
	}()

	return ch
}

func (s *Service) Wait() {
	s.stopWg.Wait()
}
