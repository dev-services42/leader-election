package main

import (
	"context"
	"fmt"
	election "github.com/dev-services42/leader-election-lib/leader-election"
	"github.com/dev-services42/leader-election-lib/leader-election/keys"
	"github.com/dev-services42/leader-election-lib/leader-election/sessions"
	config2 "github.com/dev-services42/leader-election/config"
	"github.com/dev-services42/leader-election/domain"
	"github.com/dev-services42/leader-election/facade"
	"github.com/hashicorp/consul/api"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func cmdRun(c *cli.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}

	configFile := c.String("config")
	if configFile == "" {
		configFile = "config.toml"
	}

	cfg, err := config2.Parse(configFile)
	if err != nil {
		return err
	}

	config := api.DefaultConfig()
	config.Address = cfg.Consul.Addr
	consul, err := api.NewClient(config)
	if err != nil {
		return err
	}

	srvSessions, err := sessions.New(logger, consul)
	if err != nil {
		return err
	}

	srvKeys, err := keys.New(logger, consul, cfg.Consul.GetKeyRecheckTTL())
	if err != nil {
		return err
	}

	srvLeaderElection, err := election.New(logger, consul, srvSessions, srvKeys, cfg.Consul.GetSessionTTL(), cfg.Consul.SessionName, cfg.Consul.KeyName)
	if err != nil {
		return err
	}

	srvDomain, err := domain.New(logger, srvLeaderElection, cfg.GRPC.GetSlowClientReadTTL())
	if err != nil {
		return err
	}

	if err := srvDomain.Run(ctx); err != nil {
		return err
	}

	srvFacade, err := facade.New(logger, cfg.GRPC.Listen, srvDomain)
	if err != nil {
		return err
	}

	if err := srvFacade.Run(ctx); err != nil {
		return err
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	sig := <-signalCh
	logger.Info("receive signal", zap.String("signal", sig.String()))

	signal.Stop(signalCh)
	cancel()

	srvFacade.Wait()
	srvDomain.Wait()

	return nil
}

func main() {
	a := cli.NewApp()
	a.Name = "Leader Election"
	a.Usage = "Leading election for every pod in simple way"
	a.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config,c",
			Value: "config.toml",
			Usage: "Config file path",
		},
	}
	a.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run leader election process",
			Action: cmdRun,
		},
	}

	if err := a.Run(os.Args); err != nil {
		fmt.Println("cannot exec app command", err)
	}
}
