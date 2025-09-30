package config

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string     `yaml:"env" env-required:"true" env-default:"production"`
	PGSQL      PQSQL      `yaml:"pgsql" env-required:"true"`
	HTTPServer HTTPServer `yaml:"http_server" env-required:"true"`
	JWTSecret  string     `yaml:"jwt_secret" env-required:"true" env-default:"super_secret_key"`
}

type HTTPServer struct {
	Address string `yaml:"address" env-required:"true" env-default:"localhost:8080"`
}

type PQSQL struct {
	Host     string `yaml:"host" env-required:"true" env-default:"localhost"`
	Port     string `yaml:"port" env-required:"true" env-default:"5432"`
	User     string `yaml:"user" env-required:"true" env-default:"postgres"`
	Password string `yaml:"password" env-required:"true" env-default:"password"`
	DBName   string `yaml:"dbname" env-required:"true" env-default:"stories_db"`
	SSLMode  string `yaml:"sslmode" env-required:"true" env-default:"disable"`
}

func MustLoad() *Config {
	var configPath string

	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		flags := flag.String("config", "", "Path to config file")
		flag.Parse()
		configPath = *flags

		if configPath == "" {
			log.Fatal("config path must be provided")
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist at path: %s", configPath)
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)

	if err != nil {
		log.Fatalf("failed to read config: %s", err)
	}

	return &cfg
}
