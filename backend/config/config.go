package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	File     string `mapstructure:"file"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

func (d DatabaseConfig) DSN() string {
	switch d.Type {
	case "sqlite", "sqlite3":
		f := d.File
		if f == "" {
			f = "rosetta.db"
		}
		return f
	default:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			d.User, d.Password, d.Host, d.Port, d.DBName)
	}
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./backend")
	viper.AddConfigPath("$HOME/.rosetta")

	// Environment variable overrides
	viper.SetEnvPrefix("ROSETTA")
	viper.AutomaticEnv()

	// Bind specific env vars
	_ = viper.BindEnv("database.password", "ROSETTA_DB_PASSWORD")
	_ = viper.BindEnv("jwt.secret", "ROSETTA_JWT_SECRET")
	_ = viper.BindEnv("server.port", "ROSETTA_SERVER_PORT")

	// Defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.host", "127.0.0.1")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "rosetta")
	viper.SetDefault("jwt.secret", "rosetta-jwt-secret-change-in-production")
	viper.SetDefault("jwt.expire_hours", 24)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// Config file not found, use defaults + env vars
	}

	// Override with env vars if set
	if v := os.Getenv("ROSETTA_DB_PASSWORD"); v != "" {
		viper.Set("database.password", v)
	}
	if v := os.Getenv("ROSETTA_JWT_SECRET"); v != "" {
		viper.Set("jwt.secret", v)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
