package config

import (
	"github.com/BurntSushi/toml"
	"time"
)

type GRPCConfig struct {
	Listen            string `toml:"listen"`
	SlowClientReadTTL string `toml:"slow_client_read_ttl"`
}

type ConsulConfig struct {
	Addr          string `toml:"addr"`
	SessionTTL    string `toml:"session_ttl"`
	KeyRecheckTTL string `toml:"key_recheck_ttl"`
	SessionName   string `toml:"session_name"`
	KeyName       string `toml:"key_name"`
}

type Config struct {
	Consul ConsulConfig `toml:"consul"`
	GRPC   GRPCConfig   `toml:"grpc"`
}

func (c GRPCConfig) GetSlowClientReadTTL() time.Duration {
	v, err := time.ParseDuration(c.SlowClientReadTTL)
	if err != nil {
		panic(err)
	}

	return v
}

func (c ConsulConfig) GetSessionTTL() time.Duration {
	v, err := time.ParseDuration(c.SessionTTL)
	if err != nil {
		panic(err)
	}

	return v
}

func (c ConsulConfig) GetKeyRecheckTTL() time.Duration {
	v, err := time.ParseDuration(c.KeyRecheckTTL)
	if err != nil {
		panic(err)
	}

	return v
}

func Parse(filename string) (*Config, error) {
	var c Config
	if _, err := toml.DecodeFile(filename, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
