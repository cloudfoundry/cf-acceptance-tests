package internal

import "time"

type TestSuiteConfig interface {
	GetApiEndpoint() string
	GetConfigurableTestPassword() string
	GetPersistentAppOrg() string
	GetPersistentAppQuotaName() string
	GetPersistentAppSpace() string
	GetScaledTimeout(time.Duration) time.Duration
	GetAdminPassword() string
	GetExistingUser() string
	GetExistingUserPassword() string
	GetShouldKeepUser() bool
	GetUseExistingUser() bool
	GetAdminUser() string
	GetSkipSSLValidation() bool
	GetNamePrefix() string
}
