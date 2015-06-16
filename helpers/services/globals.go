package services

import "time"

var (
	DEFAULT_TIMEOUT      = 45 * time.Second
	CF_PUSH_TIMEOUT      = 2 * time.Minute
	BROKER_START_TIMEOUT = 5 * time.Minute
)
