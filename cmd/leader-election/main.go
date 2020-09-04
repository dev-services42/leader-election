package main

import (
	"context"
	election "github.com/dev-services42/leader-election-lib/leader-election"
	"github.com/dev-services42/leader-election-lib/leader-election/keys"
	"github.com/dev-services42/leader-election-lib/leader-election/sessions"
	"github.com/dev-services42/leader-election/domain"
	"github.com/dev-services42/leader-election/facade"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	grpcAddr = ":8000"
	// disconnect slow clients
	slowClientReadTTL = time.Second
	consulAddr        = "127.0.0.1:8500"
	sessionTTL        = 60 * time.Second
	keysRecheckTTL    = 10 * time.Second
	sessionName       = "services/leader-election/leader"
	keyName           = sessionName
)

func checkErr(logger *zap.Logger, err error) {
	if err != nil {
		logger.Fatal("error during startup", zap.Error(err))
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	config := api.DefaultConfig()
	config.Address = consulAddr
	consul, err := api.NewClient(config)
	checkErr(logger, err)

	srvSessions, err := sessions.New(logger, consul)
	checkErr(logger, err)

	srvKeys, err := keys.New(logger, consul, keysRecheckTTL)
	checkErr(logger, err)

	srvLeaderElection, err := election.New(logger, consul, srvSessions, srvKeys, sessionTTL, sessionName, keyName)
	checkErr(logger, err)

	srvDomain, err := domain.New(logger, srvLeaderElection, slowClientReadTTL)
	checkErr(logger, err)

	checkErr(logger, srvDomain.Run(ctx))

	srvFacade, err := facade.New(logger, grpcAddr, srvDomain)
	checkErr(logger, err)

	checkErr(logger, srvFacade.Run(ctx))

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	sig := <-signalCh
	logger.Info("receive signal", zap.String("signal", sig.String()))

	signal.Stop(signalCh)
	cancel()

	srvFacade.Wait()
	srvDomain.Wait()
}
