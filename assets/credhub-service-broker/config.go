package main

import (
	"log"
	"os"
	"strconv"

	uuid "github.com/satori/go.uuid"
)

type CredhubConfig struct {
	API    string
	Client string
	Secret string
}

type Config struct {
	Port int

	ServiceName string
	ServiceUUID string
	PlanUUID    string

	Credhub CredhubConfig
}

func LoadConfig() Config {
	cfg := Config{
		Port:        8080,
		ServiceName: "credhub-read",
		ServiceUUID: uuid.NewV4().String(),
		PlanUUID:    uuid.NewV4().String(),
		Credhub: CredhubConfig{
			API:    os.Getenv("CREDHUB_API"),
			Client: os.Getenv("CREDHUB_CLIENT"),
			Secret: os.Getenv("CREDHUB_SECRET"),
		},
	}

	if portStr, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			log.Panicf("Invalid value for PORT: %q. Please ensure the PORT environment variable is a valid integer between 1 and 65535.", portStr)
		}
		cfg.Port = port
	}

	if serviceName, ok := os.LookupEnv("SERVICE_NAME"); ok {
		cfg.ServiceName = serviceName
	}

	if serviceUUID, ok := os.LookupEnv("SERVICE_UUID"); ok {
		cfg.ServiceUUID = serviceUUID
	}

	if planUUID, ok := os.LookupEnv("PLAN_UUID"); ok {
		cfg.PlanUUID = planUUID
	}

	if cfg.Credhub.API == "" {
		log.Panicf("Invalid value for CREDHUB_API: %q. Please ensure the CREDHUB_API environment variable is set to a valid url.", cfg.Credhub.API)
	}

	if cfg.Credhub.Client == "" {
		log.Panicf("Invalid value for CREDHUB_CLIENT: %q. Please ensure the CREDHUB_CLIENT environment variable is set to a valid CredHub client.", cfg.Credhub.Client)
	}

	if cfg.Credhub.Secret == "" {
		log.Panicf("Invalid value for CREDHUB_SECRET: %q. Please ensure the CREDHUB_SECRET environment variable is set to a valid CredHub secret for the provided client.", cfg.Credhub.Secret)
	}

	return cfg
}
