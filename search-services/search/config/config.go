package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type XKCD struct {
	URL         string        `yaml:"url" env:"XKCD_URL" env-default:"xkcd.com"`
	Concurrency int           `yaml:"concurrency" env:"XKCD_CONCURRENCY" env-default:"1"`
	Timeout     time.Duration `yaml:"timeout" env:"XKCD_TIMEOUT" env-default:"10s"`
	CheckPeriod time.Duration `yaml:"check_period" env:"XKCD_CHECK_PERIOD" env-default:"1h"`
}

type Config struct {
	LogLevel      string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address       string        `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:"localhost:80"`
	DBAddress     string        `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
	WordsAddress  string        `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"localhost:81"`
	IndexTTL      time.Duration `yaml:"index_ttl" env:"INDEX_TTL" env-default:"24h"`
	BrokerAddress string        `yaml:"broker_address" env:"BROKER_ADDRESS" env-default:"nats://nats:4222"`
	XKCD          XKCD          `yaml:"xkcd"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			log.Fatalf("cannot read config %q: %s", configPath, err)
		}
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("read env failed %q: %s", configPath, err)
	}
	return cfg
}
