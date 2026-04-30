package config

import (
	"github.com/spf13/viper"
	pkg "github.com/vhgomes/rate-guard/pkg/logging"
)

type Config struct {
	ListenAddr  string                              `mapstructure:"listen_addr"`
	MetricsAddr string                              `mapstructure:"metrics_addr"`
	Redis       RedisConfig                         `mapstructure:"redis"`
	Tenants     map[string]map[string]LimiterConfig `mapstructure:"tenants"`
}
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}
type LimiterConfig struct {
	Limit         int `mapstructure:"limit"`
	WindowSeconds int `mapstructure:"window_seconds"`
}

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		pkg.Error("Error reading config file", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		pkg.Error("Error unmarshaling config", err)
	}

	return &cfg
}
