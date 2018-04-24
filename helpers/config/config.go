package config

import (
	"time"
)

type CatsConfig interface {
	GetIncludeApps() bool
	GetIncludeBackendCompatiblity() bool
	GetIncludeContainerNetworking() bool
	GetIncludeDetect() bool
	GetIncludeDocker() bool
	GetIncludeInternetDependent() bool
	GetIncludePrivateDockerRegistry() bool
	GetIncludePersistentApp() bool
	GetIncludePrivilegedContainerSupport() bool
	GetIncludeRouteServices() bool
	GetIncludeRouting() bool
	GetIncludeZipkin() bool
	GetIncludeSSO() bool
	GetIncludeSecurityGroups() bool
	GetIncludeServices() bool
	GetIncludeSsh() bool
	GetIncludeTasks() bool
	GetIncludeV3() bool
	GetIncludeIsolationSegments() bool
	GetShouldKeepUser() bool
	GetSkipSSLValidation() bool
	GetUseExistingUser() bool

	GetIncludeNimbus() bool
	GetIncludeNimbusServiceInternalProxy() bool
	GetNimbusServiceNameInternalProxy() string
	GetIncludeNimbusNBConfig() bool
	GetIncludeNimbusNoCache() bool
	GetIncludeNimbusServicePostgres() bool
	GetNimbusServiceNamePostgres() string
	GetIncludeNimbusServiceRabbit() bool
	GetNimbusServiceNameRabbit() string
	GetNimbusServiceNameRabbit() string
	GetIncludeNimbusServiceRabbitmq() bool
	GetNimbusServicePlanRabbitmq() string
	GetNimbusServicePlanRabbitmq() string
	GetIncludeNimbusServiceRedis() bool
	GetNimbusServiceNameRedis() string
	GetIncludeNimbusServiceSCMSMongo() bool
	GetNimbusServiceNameSCMSMongo() string
	GetIncludeNimbusServiceCassandra() bool
	GetNimbusServiceNameCassandra() string
	GetNimbusServicePlanCassandra() string
	GetIncludeNimbusServiceMySQL() bool
	GetNimbusServiceNameMySQL() string
	GetIncludeNimbusServiceOCPShopRedis() bool
	GetNimbusServiceNameOCPShopRedis() string
	GetIncludeNimbusServiceProxy() bool
	GetNimbusServiceNameProxy() string
	GetIncludeNimbusServiceVault() bool
	GetNimbusServiceNameVault() string
	GetIncludeNimbusServiceSSDMRedis() bool
	GetNimbusServiceNameSSDMRedis() string
	GetIncludeNimbusServiceNFS() bool
	GetNimbusServiceNameNFS() string
	GetNimbusServicePlanNFS() string
	GetNimbusServiceNFSShare() string

	GetAdminPassword() string
	GetAdminUser() string
	GetApiEndpoint() string
	GetAppsDomain() string
	GetArtifactsDirectory() string
	GetBackend() string
	GetBinaryBuildpackName() string
	GetConfigurableTestPassword() string
	GetExistingOrganization() string
	GetUseExistingOrganization() bool
	GetExistingUser() string
	GetExistingUserPassword() string
	GetGoBuildpackName() string
	GetIsolationSegmentName() string
	GetJavaBuildpackName() string
	GetNamePrefix() string
	GetNodejsBuildpackName() string
	GetPrivateDockerRegistryImage() string
	GetPublicDockerRegistryImage() string
	GetPrivateDockerRegistryUsername() string
	GetPrivateDockerRegistryPassword() string
	GetPersistentAppHost() string
	GetPersistentAppOrg() string
	GetPersistentAppQuotaName() string
	GetPersistentAppSpace() string
	GetRubyBuildpackName() string
	Protocol() string

	AsyncServiceOperationTimeoutDuration() time.Duration
	BrokerStartTimeoutDuration() time.Duration
	CfPushTimeoutDuration() time.Duration
	DefaultTimeoutDuration() time.Duration
	DetectTimeoutDuration() time.Duration
	GetScaledTimeout(time.Duration) time.Duration
	LongCurlTimeoutDuration() time.Duration
	LongTimeoutDuration() time.Duration
	SleepTimeoutDuration() time.Duration
}

func NewCatsConfig(path string) (CatsConfig, error) {
	return NewConfig(path)
}
