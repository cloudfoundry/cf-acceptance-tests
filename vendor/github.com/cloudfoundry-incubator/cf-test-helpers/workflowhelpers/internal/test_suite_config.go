package internal

import "time"

type TestSuiteConfig interface {
	GetApiEndpoint() string
	GetScaledTimeout(time.Duration) time.Duration
	GetSkipSSLValidation() bool
	GetNamePrefix() string
}
