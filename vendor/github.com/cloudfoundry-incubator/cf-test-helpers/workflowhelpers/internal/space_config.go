package internal

import "time"

type SpaceConfig interface {
	GetScaledTimeout(time.Duration) time.Duration
	GetPersistentAppSpace() string
	GetPersistentAppOrg() string
	GetPersistentAppQuotaName() string
	GetNamePrefix() string
}
