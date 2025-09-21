package config

import "os"

type Config struct {
	Address string
}

func GetConfig() Config {
	return Config{
		Address: os.Getenv("FOLLOWER_SERVICE_ADDRESS"),
	}
}
