package services

import "time"

var (
	BROKER_START_TIMEOUT            = 5 * time.Minute
	ASYNC_SERVICE_OPERATION_TIMEOUT = 2 * time.Minute
)
