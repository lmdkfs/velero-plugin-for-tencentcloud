package main

import (
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"os"
)

const (
	regionKey                = "region"
	insecureSkipTLSVerifyKey = "insecureSkipTLSVerify"
	KIND_KEY                    = "kind"
	PERSISTENT_VOLUME_KEY       = "PersistentVolume"
	PERSISTENT_VOLUME_CLAIM_KEY = "PersistentVolumeClaim"
	MIN_REQ_VOL_SIZE_BYTES      = 10737418240
	MIN_REQ_VOL_SIZE_STRING     = "10Gi"
	TENCENT_CREDENTIALS_FILE    = "TENCENT_CREDENTIALS_FILE"
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
