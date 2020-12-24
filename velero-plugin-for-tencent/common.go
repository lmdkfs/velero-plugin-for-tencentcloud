package main

import (
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"os"
)

const (
	regionKey                = "region"
	insecureSkipTLSVerifyKey = "insecureSkipTLSVerify"
)

func loadEnv() error {
	envFile := os.Getenv("TENCENT_CREDENTIALS_FILE")
	if envFile == "" {
		return nil
	}
	if err := godotenv.Overload(envFile); err != nil {
		return errors.Wrapf(err, "error loading environment from TENCENT_CREDENTIALS_FILE (%s)", envFile)
	}
	return nil
}
