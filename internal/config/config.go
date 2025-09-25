package config

import (
	"errors"
	"os"
	"strings"

	wbfconf "github.com/wb-go/wbf/config"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Migrations MigrationsConfig `yaml:"migrations"`
	Redis      RedisConfig      `yaml:"redis"`
	Logging    LoggingConfig    `yaml:"logging"`
}

type ServerConfig struct {
	Addr               string `yaml:"addr"`
	ShutdownTimeoutSec int    `yaml:"shutdown_timeout_sec"`
	ReadTimeoutSec     int    `yaml:"read_timeout_sec"`
	WriteTimeoutSec    int    `yaml:"write_timeout_sec"`
}

type DatabaseConfig struct {
	DSN                  string `yaml:"dsn"`
	Slaves               string `yaml:"slaves"`
	MaxOpenConns         int    `yaml:"max_open_conns"`
	MaxIdleConns         int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeSec   int    `yaml:"conn_max_lifetime_sec"`
	ConnectRetries       int    `yaml:"connect_retries"`
	ConnectRetryDelaySec int    `yaml:"connect_retry_delay_sec"`
}

type MigrationsConfig struct {
	Path string `yaml:"path"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type CacheConfig struct {
	Prefix string `yaml:"prefix"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	cfgw := wbfconf.New()

	setDefaults(cfgw)

	if path == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			path = "config.yaml"
		} else if _, err := os.Stat("/app/config.yaml"); err == nil {
			path = "/app/config.yaml"
		}
	}

	if path != "" {
		if err := cfgw.Load(path); err != nil {
			if _, statErr := os.Stat(path); statErr == nil {
				return nil, err
			}
		}
	}

	var cfg Config
	if err := cfgw.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, errors.New("database.dsn is required (set in config file or DATABASE_DSN env)")
	}

	return &cfg, nil
}

func setDefaults(c *wbfconf.Config) {
	c.SetDefault("server.addr", ":8080")
	c.SetDefault("server.shutdown_timeout_sec", 15)
	c.SetDefault("server.read_timeout_sec", 10)
	c.SetDefault("server.write_timeout_sec", 10)

	c.SetDefault("database.dsn", "")
	c.SetDefault("database.slaves", "")
	c.SetDefault("database.max_open_conns", 25)
	c.SetDefault("database.max_idle_conns", 5)
	c.SetDefault("database.conn_max_lifetime_sec", 1800)
	c.SetDefault("database.connect_retries", 10)
	c.SetDefault("database.connect_retry_delay_sec", 3)

	c.SetDefault("migrations.path", "./migrations")

	c.SetDefault("redis.addr", "")
	c.SetDefault("redis.password", "")
	c.SetDefault("redis.db", 0)

	c.SetDefault("cache.prefix", "ct:")

	c.SetDefault("logging.level", "info")
}
