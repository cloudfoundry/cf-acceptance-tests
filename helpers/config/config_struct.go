package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/validationerrors"
)

type config struct {
	ApiEndpoint *string `json:"api"`
	AppsDomain  *string `json:"apps_domain"`
	UseHttp     *bool   `json:"use_http"`

	AdminPassword *string `json:"admin_password"`
	AdminUser     *string `json:"admin_user"`

	ExistingUser         *string `json:"existing_user"`
	ExistingUserPassword *string `json:"existing_user_password"`
	ShouldKeepUser       *bool   `json:"keep_user_at_suite_end"`
	UseExistingUser      *bool   `json:"use_existing_user"`

	UseExistingOrganization *bool   `json:"use_existing_organization"`
	ExistingOrganization    *string `json:"existing_organization"`

	ConfigurableTestPassword *string `json:"test_password"`

	PersistentAppHost      *string `json:"persistent_app_host"`
	PersistentAppOrg       *string `json:"persistent_app_org"`
	PersistentAppQuotaName *string `json:"persistent_app_quota_name"`
	PersistentAppSpace     *string `json:"persistent_app_space"`

	IsolationSegmentName *string `json:"isolation_segment_name"`

	Backend           *string `json:"backend"`
	SkipSSLValidation *bool   `json:"skip_ssl_validation"`

	ArtifactsDirectory *string `json:"artifacts_directory"`

	AsyncServiceOperationTimeout *int `json:"async_service_operation_timeout"`
	BrokerStartTimeout           *int `json:"broker_start_timeout"`
	CfPushTimeout                *int `json:"cf_push_timeout"`
	DefaultTimeout               *int `json:"default_timeout"`
	DetectTimeout                *int `json:"detect_timeout"`
	LongCurlTimeout              *int `json:"long_curl_timeout"`
	SleepTimeout                 *int `json:"sleep_timeout"`

	TimeoutScale *float64 `json:"timeout_scale"`

	BinaryBuildpackName     *string `json:"binary_buildpack_name"`
	GoBuildpackName         *string `json:"go_buildpack_name"`
	JavaBuildpackName       *string `json:"java_buildpack_name"`
	NodejsBuildpackName     *string `json:"nodejs_buildpack_name"`
	PhpBuildpackName        *string `json:"php_buildpack_name"`
	PythonBuildpackName     *string `json:"python_buildpack_name"`
	RubyBuildpackName       *string `json:"ruby_buildpack_name"`
	StaticFileBuildpackName *string `json:"staticfile_buildpack_name"`

	IncludeNimbus                     *bool `json:"include_nimbus"`
	IncludeNimbusServiceInternalProxy *bool `json:"include_nimbus_service_internal_proxy"`
	IncludeNimbusNBConfig             *bool `json:"include_nimbus_nb_config"`
	IncludeNimbusNoCache              *bool `json:"include_nimbus_no_cache"`
	IncludeNimbusServicePostgres      *bool `json:"include_nimbus_service_postgres"`
	IncludeNimbusServiceRabbit        *bool `json:"include_nimbus_service_rabbit"`
	IncludeNimbusServiceRabbitmq      *bool `json:"include_nimbus_service_rabbitmq"`
	IncludeNimbusServiceRedis         *bool `json:"include_nimbus_service_redis"`
	IncludeNimbusServiceSCMSMongo     *bool `json:"include_nimbus_service_scms_mongo"`
	IncludeNimbusServiceCassandra     *bool `json:"include_nimbus_service_cassandra"`
	IncludeNimbusServiceMySQL         *bool `json:"include_nimbus_service_mysql"`
	IncludeNimbusServiceOCPShopRedis  *bool `json:"include_nimbus_service_ocp_shop_redis"`
	IncludeNimbusServiceProxy         *bool `json:"include_nimbus_service_proxy"`
	IncludeNimbusServiceVault         *bool `json:"include_nimbus_service_vault"`
	IncludeNimbusServiceSSDMRedis     *bool `json:"include_nimbus_service_ssdm_redis"`
	IncludeNimbusServiceNFS           *bool `json:"include_nimbus_service_nfs"`

	NimbusServiceNameInternalProxy *string `json:"nimbus_service_name_internal_proxy"`
	NimbusServiceNamePostgres      *string `json:"nimbus_service_name_postgres"`
	NimbusServiceNameRabbit        *string `json:"nimbus_service_name_rabbit"`
	NimbusServiceNameRabbitmq      *string `json:"nimbus_service_name_rabbitmq"`
	NimbusServicePlanRabbit        *string `json:"nimbus_service_plan_rabbit"`
	NimbusServicePlanRabbitmq      *string `json:"nimbus_service_plan_rabbitmq"`
	NimbusServiceNameRedis         *string `json:"nimbus_service_name_redis"`
	NimbusServiceNameSCMSMongo     *string `json:"nimbus_service_name_scms_mongo"`
	NimbusServiceNameCassandra     *string `json:"nimbus_service_name_cassandra"`
	NimbusServicePlanCassandra     *string `json:"nimbus_service_plan_cassandra"`
	NimbusServiceNameMySQL         *string `json:"nimbus_service_name_mysql"`
	NimbusServiceNameOCPShopRedis  *string `json:"nimbus_service_name_ocp_shop_redis"`
	NimbusServiceNameProxy         *string `json:"nimbus_service_name_proxy"`
	NimbusServiceNameVault         *string `json:"nimbus_service_name_vault"`
	NimbusServiceNameSSDMRedis     *string `json:"nimbus_service_name_ssdm_redis"`
	NimbusServiceNameNFS           *string `json:"nimbus_service_name_nfs"`
	NimbusServicePlanNFS           *string `json:"nimbus_service_plan_nfs"`
	NimbusServiceNFSShare          *string `json:"nimbus_service_nfs_share"`

	IncludeApps                       *bool `json:"include_apps"`
	IncludeBackendCompatiblity        *bool `json:"include_backend_compatibility"`
	IncludeContainerNetworking        *bool `json:"include_container_networking"`
	IncludeDetect                     *bool `json:"include_detect"`
	IncludeDocker                     *bool `json:"include_docker"`
	IncludeInternetDependent          *bool `json:"include_internet_dependent"`
	IncludePersistentApp              *bool `json:"include_persistent_app"`
	IncludePrivateDockerRegistry      *bool `json:"include_private_docker_registry"`
	IncludePrivilegedContainerSupport *bool `json:"include_privileged_container_support"`
	IncludeRouteServices              *bool `json:"include_route_services"`
	IncludeRouting                    *bool `json:"include_routing"`
	IncludeSSO                        *bool `json:"include_sso"`
	IncludeSecurityGroups             *bool `json:"include_security_groups"`
	IncludeServices                   *bool `json:"include_services"`
	IncludeSsh                        *bool `json:"include_ssh"`
	IncludeTasks                      *bool `json:"include_tasks"`
	IncludeV3                         *bool `json:"include_v3"`
	IncludeZipkin                     *bool `json:"include_zipkin"`
	IncludeIsolationSegments          *bool `json:"include_isolation_segments"`

	PrivateDockerRegistryImage    *string `json:"private_docker_registry_image"`
	PublicDockerRegistryImage     *string `json:"public_docker_registry_image"`
	PrivateDockerRegistryUsername *string `json:"private_docker_registry_username"`
	PrivateDockerRegistryPassword *string `json:"private_docker_registry_password"`

	NamePrefix *string `json:"name_prefix"`
}

var defaults = config{}

func ptrToString(str string) *string {
	return &str
}

func ptrToBool(b bool) *bool {
	return &b
}

func ptrToInt(i int) *int {
	return &i
}

func ptrToFloat(f float64) *float64 {
	return &f
}

func getDefaults() config {
	defaults.Backend = ptrToString("")
	defaults.PersistentAppHost = ptrToString("CATS-persistent-app")

	defaults.PersistentAppOrg = ptrToString("CATS-persistent-org")
	defaults.PersistentAppQuotaName = ptrToString("CATS-persistent-quota")
	defaults.PersistentAppSpace = ptrToString("CATS-persistent-space")

	defaults.IsolationSegmentName = ptrToString("")

	defaults.BinaryBuildpackName = ptrToString("binary_buildpack")
	defaults.GoBuildpackName = ptrToString("go_buildpack")
	defaults.JavaBuildpackName = ptrToString("java_buildpack")
	defaults.NodejsBuildpackName = ptrToString("nodejs_buildpack")
	defaults.PhpBuildpackName = ptrToString("php_buildpack")
	defaults.PythonBuildpackName = ptrToString("python_buildpack")
	defaults.RubyBuildpackName = ptrToString("ruby_buildpack")
	defaults.StaticFileBuildpackName = ptrToString("staticfile_buildpack")

	defaults.IncludeApps = ptrToBool(true)
	defaults.IncludeDetect = ptrToBool(true)
	defaults.IncludePersistentApp = ptrToBool(true)
	defaults.IncludeRouting = ptrToBool(true)

	defaults.IncludeBackendCompatiblity = ptrToBool(false)
	defaults.IncludeContainerNetworking = ptrToBool(false)
	defaults.IncludeDocker = ptrToBool(false)
	defaults.IncludeInternetDependent = ptrToBool(false)
	defaults.IncludeIsolationSegments = ptrToBool(false)
	defaults.IncludePrivilegedContainerSupport = ptrToBool(false)
	defaults.IncludePrivateDockerRegistry = ptrToBool(false)
	defaults.IncludeRouteServices = ptrToBool(false)
	defaults.IncludeSSO = ptrToBool(false)
	defaults.IncludeSecurityGroups = ptrToBool(false)
	defaults.IncludeServices = ptrToBool(false)
	defaults.IncludeSsh = ptrToBool(false)
	defaults.IncludeTasks = ptrToBool(false)
	defaults.IncludeV3 = ptrToBool(false)
	defaults.IncludeZipkin = ptrToBool(false)

	defaults.UseHttp = ptrToBool(false)
	defaults.UseExistingUser = ptrToBool(false)
	defaults.ShouldKeepUser = ptrToBool(false)

	defaults.UseExistingOrganization = ptrToBool(false)
	defaults.ExistingOrganization = ptrToString("")

	defaults.AsyncServiceOperationTimeout = ptrToInt(2)
	defaults.BrokerStartTimeout = ptrToInt(5)
	defaults.CfPushTimeout = ptrToInt(2)
	defaults.DefaultTimeout = ptrToInt(30)
	defaults.DetectTimeout = ptrToInt(5)
	defaults.LongCurlTimeout = ptrToInt(2)
	defaults.SleepTimeout = ptrToInt(30)

	defaults.ConfigurableTestPassword = ptrToString("")

	defaults.TimeoutScale = ptrToFloat(2.0)

	defaults.ArtifactsDirectory = ptrToString(filepath.Join("..", "results"))

	defaults.PrivateDockerRegistryImage = ptrToString("")
	defaults.PublicDockerRegistryImage = ptrToString("cloudfoundry/diego-docker-app-custom:latest")
	defaults.PrivateDockerRegistryUsername = ptrToString("")
	defaults.PrivateDockerRegistryPassword = ptrToString("")

	defaults.IncludeNimbus = ptrToBool(true)
	defaults.IncludeNimbusServiceInternalProxy = ptrToBool(false)
	defaults.IncludeNimbusNBConfig = ptrToBool(false)
	defaults.IncludeNimbusNoCache = ptrToBool(false)
	defaults.IncludeNimbusServicePostgres = ptrToBool(false)
	defaults.IncludeNimbusServiceRabbit = ptrToBool(false)
	defaults.IncludeNimbusServiceRabbitmq = ptrToBool(false)
	defaults.IncludeNimbusServiceRedis = ptrToBool(false)
	defaults.IncludeNimbusServiceSCMSMongo = ptrToBool(false)
	defaults.IncludeNimbusServiceCassandra = ptrToBool(false)
	defaults.IncludeNimbusServiceMySQL = ptrToBool(false)
	defaults.IncludeNimbusServiceOCPShopRedis = ptrToBool(false)
	defaults.IncludeNimbusServiceProxy = ptrToBool(false)
	defaults.IncludeNimbusServiceVault = ptrToBool(false)
	defaults.IncludeNimbusServiceSSDMRedis = ptrToBool(false)
	defaults.IncludeNimbusServiceNFS = ptrToBool(false)

	defaults.NimbusServiceNameInternalProxy = ptrToString("internal-proxy")
	defaults.NimbusServiceNamePostgres = ptrToString("postgresql94")
	defaults.NimbusServiceNameRabbit = ptrToString("l2-rabbitmq")
	defaults.NimbusServiceNameRabbitmq = ptrToString("rabbitmq")
	defaults.NimbusServicePlanRabbit = ptrToString("default")
	defaults.NimbusServicePlanRabbitmq = ptrToString("standard")
	defaults.NimbusServiceNameRedis = ptrToString("redis")
	defaults.NimbusServiceNameSCMSMongo = ptrToString("scms-mongo3")
	defaults.NimbusServiceNameCassandra = ptrToString("cassandra")
	defaults.NimbusServicePlanCassandra = ptrToString("default")
	defaults.NimbusServiceNameMySQL = ptrToString("l2-mysql")
	defaults.NimbusServiceNameOCPShopRedis = ptrToString("ocpshop-redis")
	defaults.NimbusServiceNameProxy = ptrToString("proxy")
	defaults.NimbusServiceNameVault = ptrToString("vault-ha")
	defaults.NimbusServiceNameNFS = ptrToString("nfs")
	defaults.NimbusServicePlanNFS = ptrToString("Existing")
	defaults.NimbusServiceNFSShare = ptrToString("")
	defaults.NimbusServiceNameSSDMRedis = ptrToString("ssdm-redis")

	defaults.NamePrefix = ptrToString("CATS")
	return defaults
}

func NewConfig(path string) (*config, error) {
	d := getDefaults()
	cfg := &d
	err := load(path, cfg)
	if err.Empty() {
		return cfg, nil
	}
	return nil, err
}

func validateConfig(config *config) Errors {
	errs := Errors{}

	var err error
	err = validateAdminUser(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateAdminPassword(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateApiEndpoint(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateAppsDomain(config)
	if err != nil {
		errs.Add(err)
	}

	err = validateBackend(config)
	if err != nil {
		errs.Add(err)
	}

	err = validatePrivateDockerRegistry(config)
	if err != nil {
		errs.Add(err)
	}

	if config.UseHttp == nil {
		errs.Add(fmt.Errorf("* 'use_http' must not be null"))
	}
	if config.ShouldKeepUser == nil {
		errs.Add(fmt.Errorf("* 'keep_user_at_suite_end' must not be null"))
	}
	if config.UseExistingUser == nil {
		errs.Add(fmt.Errorf("* 'use_existing_user' must not be null"))
	}
	if config.ConfigurableTestPassword == nil {
		errs.Add(fmt.Errorf("* 'test_password' must not be null"))
	}
	if config.PersistentAppHost == nil {
		errs.Add(fmt.Errorf("* 'persistent_app_host' must not be null"))
	}
	if config.PersistentAppOrg == nil {
		errs.Add(fmt.Errorf("* 'persistent_app_org' must not be null"))
	}
	if config.PersistentAppQuotaName == nil {
		errs.Add(fmt.Errorf("* 'persistent_app_quota_name' must not be null"))
	}
	if config.PersistentAppSpace == nil {
		errs.Add(fmt.Errorf("* 'persistent_app_space' must not be null"))
	}
	if config.IsolationSegmentName == nil {
		errs.Add(fmt.Errorf("* 'isolation_segment_name' must not be null"))
	}
	if config.SkipSSLValidation == nil {
		errs.Add(fmt.Errorf("* 'skip_ssl_validation' must not be null"))
	}
	if config.ArtifactsDirectory == nil {
		errs.Add(fmt.Errorf("* 'artifacts_directory' must not be null"))
	}
	if config.AsyncServiceOperationTimeout == nil {
		errs.Add(fmt.Errorf("* 'async_service_operation_timeout' must not be null"))
	}
	if config.BrokerStartTimeout == nil {
		errs.Add(fmt.Errorf("* 'broker_start_timeout' must not be null"))
	}
	if config.CfPushTimeout == nil {
		errs.Add(fmt.Errorf("* 'cf_push_timeout' must not be null"))
	}
	if config.DefaultTimeout == nil {
		errs.Add(fmt.Errorf("* 'default_timeout' must not be null"))
	}
	if config.DetectTimeout == nil {
		errs.Add(fmt.Errorf("* 'detect_timeout' must not be null"))
	}
	if config.LongCurlTimeout == nil {
		errs.Add(fmt.Errorf("* 'long_curl_timeout' must not be null"))
	}
	if config.SleepTimeout == nil {
		errs.Add(fmt.Errorf("* 'sleep_timeout' must not be null"))
	}
	if config.TimeoutScale == nil {
		errs.Add(fmt.Errorf("* 'timeout_scale' must not be null"))
	}
	if config.BinaryBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'binary_buildpack_name' must not be null"))
	}
	if config.GoBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'go_buildpack_name' must not be null"))
	}
	if config.JavaBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'java_buildpack_name' must not be null"))
	}
	if config.NodejsBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'nodejs_buildpack_name' must not be null"))
	}
	if config.PhpBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'php_buildpack_name' must not be null"))
	}
	if config.PythonBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'python_buildpack_name' must not be null"))
	}
	if config.RubyBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'ruby_buildpack_name' must not be null"))
	}
	if config.StaticFileBuildpackName == nil {
		errs.Add(fmt.Errorf("* 'staticfile_buildpack_name' must not be null"))
	}
	if config.IncludeApps == nil {
		errs.Add(fmt.Errorf("* 'include_apps' must not be null"))
	}
	if config.IncludeNimbus == nil {
		errs.Add(fmt.Errorf("* 'include_nimbus' must not be null"))
	}
	if config.IncludeBackendCompatiblity == nil {
		errs.Add(fmt.Errorf("* 'include_backend_compatibility' must not be null"))
	}
	if config.IncludeContainerNetworking == nil {
		errs.Add(fmt.Errorf("* 'include_container_networking' must not be null"))
	}
	if config.IncludeDetect == nil {
		errs.Add(fmt.Errorf("* 'include_detect' must not be null"))
	}
	if config.IncludeDocker == nil {
		errs.Add(fmt.Errorf("* 'include_docker' must not be null"))
	}
	if config.IncludeInternetDependent == nil {
		errs.Add(fmt.Errorf("* 'include_internet_dependent' must not be null"))
	}
	if config.IncludePrivateDockerRegistry == nil {
		errs.Add(fmt.Errorf("* 'include_private_docker_registry' must not be null"))
	}
	if config.IncludePersistentApp == nil {
		errs.Add(fmt.Errorf("* 'include_persistent_app' must not be null"))
	}
	if config.IncludePrivilegedContainerSupport == nil {
		errs.Add(fmt.Errorf("* 'include_privileged_container_support' must not be null"))
	}
	if config.IncludeRouteServices == nil {
		errs.Add(fmt.Errorf("* 'include_route_services' must not be null"))
	}
	if config.IncludeRouting == nil {
		errs.Add(fmt.Errorf("* 'include_routing' must not be null"))
	}
	if config.IncludeSSO == nil {
		errs.Add(fmt.Errorf("* 'include_sso' must not be null"))
	}
	if config.IncludeSecurityGroups == nil {
		errs.Add(fmt.Errorf("* 'include_security_groups' must not be null"))
	}
	if config.IncludeServices == nil {
		errs.Add(fmt.Errorf("* 'include_services' must not be null"))
	}
	if config.IncludeSsh == nil {
		errs.Add(fmt.Errorf("* 'include_ssh' must not be null"))
	}
	if config.IncludeTasks == nil {
		errs.Add(fmt.Errorf("* 'include_tasks' must not be null"))
	}
	if config.IncludeV3 == nil {
		errs.Add(fmt.Errorf("* 'include_v3' must not be null"))
	}
	if config.IncludeZipkin == nil {
		errs.Add(fmt.Errorf("* 'include_zipkin' must not be null"))
	}
	if config.IncludeIsolationSegments == nil {
		errs.Add(fmt.Errorf("* 'include_isolation_segments' must not be null"))
	}
	if config.PrivateDockerRegistryImage == nil {
		errs.Add(fmt.Errorf("* 'private_docker_registry_image' must not be null"))
	}
	if config.PrivateDockerRegistryUsername == nil {
		errs.Add(fmt.Errorf("* 'private_docker_registry_username' must not be null"))
	}
	if config.PrivateDockerRegistryPassword == nil {
		errs.Add(fmt.Errorf("* 'private_docker_registry_password' must not be null"))
	}
	if config.NamePrefix == nil {
		errs.Add(fmt.Errorf("* 'name_prefix' must not be null"))
	}
	if *config.IncludeNimbusServiceNFS == true && *config.NimbusServiceNFSShare == "" {
		errs.Add(fmt.Errorf("* 'nimbus_service_nfs_share' must not be null"))
	}

	return errs
}

func validateBackend(config *config) error {
	if config.Backend == nil {
		return fmt.Errorf("* 'backend' must not be null")
	}

	if config.GetBackend() != "dea" && config.GetBackend() != "diego" && config.GetBackend() != "" {
		return fmt.Errorf("* Invalid configuration: 'backend' must be 'diego', 'dea', or empty but was set to '%s'", config.GetBackend())
	}

	return nil
}

func validateApiEndpoint(config *config) error {
	if config.ApiEndpoint == nil {
		return fmt.Errorf("* 'api' must not be null")
	}

	if config.GetApiEndpoint() == "" {
		return fmt.Errorf("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank")
	}

	u, err := url.Parse(config.GetApiEndpoint())
	if err != nil {
		return fmt.Errorf("* Invalid configuration: 'api' must be a valid URL but was set to '%s'", config.GetApiEndpoint())
	}

	host := u.Host
	if host == "" {
		// url.Parse misunderstood our convention and treated the hostname as a URL path
		host = u.Path
	}

	if _, err = net.LookupHost(host); err != nil {
		return fmt.Errorf("* Invalid configuration for 'api' <%s>: %s", config.GetApiEndpoint(), err)
	}

	return nil
}

func validateAppsDomain(config *config) error {
	if config.AppsDomain == nil {
		return fmt.Errorf("* 'apps_domain' must not be null")
	}

	madeUpAppHostname := "made-up-app-host-name." + config.GetAppsDomain()
	u, err := url.Parse(madeUpAppHostname)
	if err != nil {
		return fmt.Errorf("* Invalid configuration: 'apps_domain' must be a valid URL but was set to '%s'", config.GetAppsDomain())
	}

	host := u.Host
	if host == "" {
		// url.Parse misunderstood our convention and treated the hostname as a URL path
		host = u.Path
	}

	if _, err = net.LookupHost(madeUpAppHostname); err != nil {
		return fmt.Errorf("* Invalid configuration for 'apps_domain' <%s>: %s", config.GetAppsDomain(), err)
	}

	return nil
}

func validateAdminUser(config *config) error {
	if config.AdminUser == nil {
		return fmt.Errorf("* 'admin_user' must not be null")
	}

	if config.GetAdminUser() == "" {
		return fmt.Errorf("* Invalid configuration: 'admin_user' must be provided")
	}

	return nil
}

func validateAdminPassword(config *config) error {
	if config.AdminPassword == nil {
		return fmt.Errorf("* 'admin_password' must not be null")
	}

	if config.GetAdminPassword() == "" {
		return fmt.Errorf("* Invalid configuration: 'admin_password' must be provided")
	}

	return nil
}

func validatePrivateDockerRegistry(config *config) error {
	if config.IncludePrivateDockerRegistry == nil {
		return fmt.Errorf("* 'include_private_docker_registry' must not be null")
	}
	if config.PrivateDockerRegistryImage == nil {
		return fmt.Errorf("* 'private_docker_registry_image' must not be null")
	}
	if config.PrivateDockerRegistryUsername == nil {
		return fmt.Errorf("* 'private_docker_registry_username' must not be null")
	}
	if config.PrivateDockerRegistryPassword == nil {
		return fmt.Errorf("* 'private_docker_registry_password' must not be null")
	}

	if !config.GetIncludePrivateDockerRegistry() {
		return nil
	}

	if config.GetPrivateDockerRegistryImage() == "" {
		return fmt.Errorf("* Invalid configuration: 'private_docker_registry_image' must be provided if 'include_private_docker_registry' is true")
	}
	if config.GetPrivateDockerRegistryUsername() == "" {
		return fmt.Errorf("* Invalid configuration: 'private_docker_registry_username' must be provided if 'include_private_docker_registry' is true")
	}
	if config.GetPrivateDockerRegistryPassword() == "" {
		return fmt.Errorf("* Invalid configuration: 'private_docker_registry_password' must be provided if 'include_private_docker_registry' is true")
	}

	return nil
}

func load(path string, config *config) Errors {
	errs := Errors{}
	err := loadConfigFromPath(path, config)
	if err != nil {
		errs.Add(fmt.Errorf("* Failed to unmarshal: %s", err))
		return errs
	}

	errs = validateConfig(config)
	if !errs.Empty() {
		return errs
	}

	if *config.TimeoutScale <= 0 {
		*config.TimeoutScale = 1.0
	}

	return errs
}

func loadConfigFromPath(path string, config interface{}) error {
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	return decoder.Decode(config)
}

func (c config) GetScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * *c.TimeoutScale)
}

func (c *config) DefaultTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.DefaultTimeout) * time.Second)
}

func (c *config) LongTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.DefaultTimeout) * time.Second)
}

func (c *config) LongCurlTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.LongCurlTimeout) * time.Minute)
}

func (c *config) SleepTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.SleepTimeout) * time.Second)
}

func (c *config) DetectTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.DetectTimeout) * time.Minute)
}

func (c *config) CfPushTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.CfPushTimeout) * time.Minute)
}

func (c *config) BrokerStartTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.BrokerStartTimeout) * time.Minute)
}

func (c *config) AsyncServiceOperationTimeoutDuration() time.Duration {
	return c.GetScaledTimeout(time.Duration(*c.AsyncServiceOperationTimeout) * time.Minute)
}

func (c *config) Protocol() string {
	if *c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func (c *config) GetAppsDomain() string {
	return *c.AppsDomain
}

func (c *config) GetSkipSSLValidation() bool {
	return *c.SkipSSLValidation
}

func (c *config) GetArtifactsDirectory() string {
	return *c.ArtifactsDirectory
}

func (c *config) GetPersistentAppSpace() string {
	return *c.PersistentAppSpace
}
func (c *config) GetPersistentAppOrg() string {
	return *c.PersistentAppOrg
}
func (c *config) GetPersistentAppQuotaName() string {
	return *c.PersistentAppQuotaName
}

func (c *config) GetIsolationSegmentName() string {
	return *c.IsolationSegmentName
}

func (c *config) GetNamePrefix() string {
	return *c.NamePrefix
}

func (c *config) GetExistingOrganization() string {
	return *c.ExistingOrganization
}

func (c *config) GetUseExistingOrganization() bool {
	return *c.UseExistingOrganization
}

func (c *config) GetUseExistingUser() bool {
	return *c.UseExistingUser
}

func (c *config) GetExistingUser() string {
	return *c.ExistingUser
}

func (c *config) GetExistingUserPassword() string {
	return *c.ExistingUserPassword
}

func (c *config) GetConfigurableTestPassword() string {
	return *c.ConfigurableTestPassword
}

func (c *config) GetShouldKeepUser() bool {
	return *c.ShouldKeepUser
}

func (c *config) GetAdminUser() string {
	return *c.AdminUser
}

func (c *config) GetAdminPassword() string {
	return *c.AdminPassword
}

func (c *config) GetApiEndpoint() string {
	return *c.ApiEndpoint
}

func (c *config) GetIncludeSsh() bool {
	return *c.IncludeSsh
}

func (c *config) GetIncludeApps() bool {
	return *c.IncludeApps
}

func (c *config) GetIncludePersistentApp() bool {
	return *c.IncludePersistentApp
}

func (c *config) GetIncludeBackendCompatiblity() bool {
	return *c.IncludeBackendCompatiblity
}

func (c *config) GetIncludeContainerNetworking() bool {
	return *c.IncludeContainerNetworking
}

func (c *config) GetIncludeDetect() bool {
	return *c.IncludeDetect
}

func (c *config) GetIncludeDocker() bool {
	return *c.IncludeDocker
}

func (c *config) GetIncludeInternetDependent() bool {
	return *c.IncludeInternetDependent
}

func (c *config) GetIncludeRouteServices() bool {
	return *c.IncludeRouteServices
}

func (c *config) GetIncludeRouting() bool {
	return *c.IncludeRouting
}

func (c *config) GetIncludeZipkin() bool {
	return *c.IncludeZipkin
}

func (c *config) GetIncludeTasks() bool {
	return *c.IncludeTasks
}

func (c *config) GetIncludePrivateDockerRegistry() bool {
	return *c.IncludePrivateDockerRegistry
}

func (c *config) GetIncludePrivilegedContainerSupport() bool {
	return *c.IncludePrivilegedContainerSupport
}

func (c *config) GetIncludeSecurityGroups() bool {
	return *c.IncludeSecurityGroups
}

func (c *config) GetIncludeServices() bool {
	return *c.IncludeServices
}

func (c *config) GetIncludeSSO() bool {
	return *c.IncludeSSO
}

func (c *config) GetIncludeV3() bool {
	return *c.IncludeV3
}

func (c *config) GetIncludeIsolationSegments() bool {
	return *c.IncludeIsolationSegments
}

func (c *config) GetIncludeNimbus() bool {
	return *c.IncludeNimbus
}

func (c *config) GetIncludeNimbusServiceInternalProxy() bool {
	return *c.IncludeNimbusServiceInternalProxy
}

func (c *config) GetNimbusServiceNameInternalProxy() string {
	return *c.NimbusServiceNameInternalProxy
}

func (c *config) GetIncludeNimbusNBConfig() bool {
	return *c.IncludeNimbusNBConfig
}

func (c *config) GetIncludeNimbusNoCache() bool {
	return *c.IncludeNimbusNoCache
}

func (c *config) GetIncludeNimbusServicePostgres() bool {
	return *c.IncludeNimbusServicePostgres
}

func (c *config) GetNimbusServiceNamePostgres() string {
	return *c.NimbusServiceNamePostgres
}

func (c *config) GetIncludeNimbusServiceRabbit() bool {
	return *c.IncludeNimbusServiceRabbit
}

func (c *config) GetNimbusServiceNameRabbit() string {
	return *c.NimbusServiceNameRabbit
}

func (c *config) GetNimbusServiceNameRabbitmq() string {
	return *c.NimbusServiceNameRabbitmq
}

func (c *config) GetNimbusServicePlanRabbitmq() string {
	return *c.NimbusServicePlanRabbitmq
}

func (c *config) GetIncludeNimbusServiceRedis() bool {
	return *c.IncludeNimbusServiceRedis
}

func (c *config) GetNimbusServiceNameRedis() string {
	return *c.NimbusServiceNameRedis
}

func (c *config) GetIncludeNimbusServiceSCMSMongo() bool {
	return *c.IncludeNimbusServiceSCMSMongo
}

func (c *config) GetNimbusServiceNameSCMSMongo() string {
	return *c.NimbusServiceNameSCMSMongo
}

func (c *config) GetIncludeNimbusServiceCassandra() bool {
	return *c.IncludeNimbusServiceCassandra
}

func (c *config) GetNimbusServiceNameCassandra() string {
	return *c.NimbusServiceNameCassandra
}

func (c *config) GetNimbusServicePlanCassandra() string {
	return *c.NimbusServicePlanCassandra
}

func (c *config) GetIncludeNimbusServiceMySQL() bool {
	return *c.IncludeNimbusServiceMySQL
}

func (c *config) GetNimbusServiceNameMySQL() string {
	return *c.NimbusServiceNameMySQL
}

func (c *config) GetIncludeNimbusServiceOCPShopRedis() bool {
	return *c.IncludeNimbusServiceOCPShopRedis
}

func (c *config) GetNimbusServiceNameOCPShopRedis() string {
	return *c.NimbusServiceNameOCPShopRedis
}

func (c *config) GetIncludeNimbusServiceProxy() bool {
	return *c.IncludeNimbusServiceProxy
}

func (c *config) GetNimbusServiceNameProxy() string {
	return *c.NimbusServiceNameProxy
}

func (c *config) GetIncludeNimbusServiceVault() bool {
	return *c.IncludeNimbusServiceVault
}

func (c *config) GetNimbusServiceNameVault() string {
	return *c.NimbusServiceNameVault
}

func (c *config) GetIncludeNimbusServiceSSDMRedis() bool {
	return *c.IncludeNimbusServiceSSDMRedis
}

func (c *config) GetNimbusServiceNameSSDMRedis() string {
	return *c.NimbusServiceNameSSDMRedis
}

func (c *config) GetIncludeNimbusServiceNFS() bool {
	return *c.IncludeNimbusServiceNFS
}

func (c *config) GetNimbusServiceNameNFS() string {
	return *c.NimbusServiceNameNFS
}

func (c *config) GetNimbusServicePlanNFS() string {
	return *c.NimbusServicePlanNFS
}

func (c *config) GetNimbusServiceNFSShare() string {
	return *c.NimbusServiceNFSShare
}

func (c *config) GetRubyBuildpackName() string {
	return *c.RubyBuildpackName
}

func (c *config) GetGoBuildpackName() string {
	return *c.GoBuildpackName
}

func (c *config) GetJavaBuildpackName() string {
	return *c.JavaBuildpackName
}

func (c *config) GetNodejsBuildpackName() string {
	return *c.NodejsBuildpackName
}

func (c *config) GetBinaryBuildpackName() string {
	return *c.BinaryBuildpackName
}

func (c *config) GetPersistentAppHost() string {
	return *c.PersistentAppHost
}

func (c *config) GetBackend() string {
	return *c.Backend
}

func (c *config) GetPrivateDockerRegistryImage() string {
	return *c.PrivateDockerRegistryImage
}

func (c *config) GetPrivateDockerRegistryUsername() string {
	return *c.PrivateDockerRegistryUsername
}

func (c *config) GetPrivateDockerRegistryPassword() string {
	return *c.PrivateDockerRegistryPassword
}

func (c *config) GetPublicDockerRegistryImage() string {
	return *c.PublicDockerRegistryImage
}
