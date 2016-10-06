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
	ApiEndpoint string `json:"api"`
	AppsDomain  string `json:"apps_domain"`
	UseHttp     bool   `json:"use_http"`

	AdminPassword string `json:"admin_password"`
	AdminUser     string `json:"admin_user"`

	ExistingUser         string `json:"existing_user"`
	ExistingUserPassword string `json:"existing_user_password"`
	ShouldKeepUser       bool   `json:"keep_user_at_suite_end"`
	UseExistingUser      bool   `json:"use_existing_user"`

	ConfigurableTestPassword string `json:"test_password"`

	PersistentAppHost      string `json:"persistent_app_host"`
	PersistentAppOrg       string `json:"persistent_app_org"`
	PersistentAppQuotaName string `json:"persistent_app_quota_name"`
	PersistentAppSpace     string `json:"persistent_app_space"`

	Backend           string `json:"backend"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`

	ArtifactsDirectory string `json:"artifacts_directory"`

	AsyncServiceOperationTimeout int `json:"async_service_operation_timeout"`
	BrokerStartTimeout           int `json:"broker_start_timeout"`
	CfPushTimeout                int `json:"cf_push_timeout"`
	DefaultTimeout               int `json:"default_timeout"`
	DetectTimeout                int `json:"detect_timeout"`
	LongCurlTimeout              int `json:"long_curl_timeout"`
	SleepTimeout                 int `json:"sleep_timeout"`

	TimeoutScale float64 `json:"timeout_scale"`

	BinaryBuildpackName     string `json:"binary_buildpack_name"`
	GoBuildpackName         string `json:"go_buildpack_name"`
	JavaBuildpackName       string `json:"java_buildpack_name"`
	NodejsBuildpackName     string `json:"nodejs_buildpack_name"`
	PhpBuildpackName        string `json:"php_buildpack_name"`
	PythonBuildpackName     string `json:"python_buildpack_name"`
	RubyBuildpackName       string `json:"ruby_buildpack_name"`
	StaticFileBuildpackName string `json:"staticfile_buildpack_name"`

	IncludeApps                       bool `json:"include_apps"`
	IncludeBackendCompatiblity        bool `json:"include_backend_compatibility"`
	IncludeDetect                     bool `json:"include_detect"`
	IncludeDocker                     bool `json:"include_docker"`
	IncludeInternetDependent          bool `json:"include_internet_dependent"`
	IncludePrivilegedContainerSupport bool `json:"include_privileged_container_support"`
	IncludeRouteServices              bool `json:"include_route_services"`
	IncludeRouting                    bool `json:"include_routing"`
	IncludeSSO                        bool `json:"include_sso"`
	IncludeSecurityGroups             bool `json:"include_security_groups"`
	IncludeServices                   bool `json:"include_services"`
	IncludeSsh                        bool `json:"include_ssh"`
	IncludeTasks                      bool `json:"include_tasks"`
	IncludeV3                         bool `json:"include_v3"`

	NamePrefix string `json:"name_prefix"`
}

var defaults = config{
	PersistentAppHost:      "CATS-persistent-app",
	PersistentAppOrg:       "CATS-persistent-org",
	PersistentAppQuotaName: "CATS-persistent-quota",
	PersistentAppSpace:     "CATS-persistent-space",

	BinaryBuildpackName:     "binary_buildpack",
	GoBuildpackName:         "go_buildpack",
	JavaBuildpackName:       "java_buildpack",
	NodejsBuildpackName:     "nodejs_buildpack",
	PhpBuildpackName:        "php_buildpack",
	PythonBuildpackName:     "python_buildpack",
	RubyBuildpackName:       "ruby_buildpack",
	StaticFileBuildpackName: "staticfile_buildpack",

	IncludeApps:                true,
	IncludeBackendCompatiblity: true,
	IncludeDetect:              true,
	IncludeDocker:              true,
	IncludeInternetDependent:   true,
	IncludeRouteServices:       true,
	IncludeRouting:             true,
	IncludeSecurityGroups:      true,
	IncludeServices:            true,
	IncludeSsh:                 true,
	IncludeV3:                  true,

	AsyncServiceOperationTimeout: 2,
	BrokerStartTimeout:           5,
	CfPushTimeout:                2,
	DefaultTimeout:               30,
	DetectTimeout:                5,
	LongCurlTimeout:              2,
	SleepTimeout:                 30,

	ArtifactsDirectory: filepath.Join("..", "results"),

	NamePrefix: "CATS",
}

var cfg *config

func NewConfig() (*config, error) {
	cfg = &defaults
	err := load(configPath(), cfg)
	if err.Empty() {
		return cfg, nil
	}
	return nil, err
}

func load(path string, config *config) Errors {
	errs := Errors{}
	err := loadConfigFromPath(path, config)
	if err != nil {
		errs.Add(err)
	}

	if config.ApiEndpoint == "" {
		errs.Add(fmt.Errorf("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank"))
	}

	var u *url.URL
	var host string
	if u, err = url.Parse(config.ApiEndpoint); err != nil {
		errs.Add(fmt.Errorf("* Invalid configuration: 'api' must be a valid URL but was set to '%s'", config.ApiEndpoint))
	} else {
		host = u.Host
		if host == "" {
			// url.Parse misunderstood our convention and treated the hostname as a URL path
			host = u.Path
		}

		if _, err = net.LookupHost(host); err != nil {
			errs.Add(fmt.Errorf("* Invalid configuration for ApiEndpoint <%s> (host %s): %s", config.ApiEndpoint, host, err))
		}
	}

	madeUpAppHostname := "made-up-hostname-that-will-never-resolve." + config.AppsDomain
	if _, err = net.LookupHost(madeUpAppHostname); err != nil {
		errs.Add(fmt.Errorf("* Invalid configuration for AppDomain <%s> (host %s): %s", config.AppsDomain, host, err))
	}

	if config.AdminUser == "" {
		errs.Add(fmt.Errorf("* Invalid configuration: 'admin_user' must be provided"))
	}

	if config.AdminPassword == "" {
		errs.Add(fmt.Errorf("* Invalid configuration: 'admin_password' must be provided"))
	}

	if config.Backend != "dea" && config.Backend != "diego" && config.Backend != "" {
		errs.Add(fmt.Errorf("* Invalid configuration: 'backend' must be 'diego', 'dea', or empty but was set to '%s'", config.Backend))
	}

	if config.TimeoutScale <= 0 {
		config.TimeoutScale = 1.0
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

func configPath() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}

func (c config) GetScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * c.TimeoutScale)
}

func (c *config) DefaultTimeoutDuration() time.Duration {
	return time.Duration(c.DefaultTimeout) * time.Second
}

func (c *config) LongTimeoutDuration() time.Duration {
	return time.Duration(c.DefaultTimeout) * time.Second
}

func (c *config) LongCurlTimeoutDuration() time.Duration {
	return time.Duration(c.LongCurlTimeout) * time.Minute
}

func (c *config) SleepTimeoutDuration() time.Duration {
	return time.Duration(c.SleepTimeout) * time.Second
}

func (c *config) DetectTimeoutDuration() time.Duration {
	return time.Duration(c.DetectTimeout) * time.Minute
}

func (c *config) CfPushTimeoutDuration() time.Duration {
	return time.Duration(c.CfPushTimeout) * time.Minute
}

func (c *config) BrokerStartTimeoutDuration() time.Duration {
	return time.Duration(c.BrokerStartTimeout) * time.Minute
}

func (c *config) AsyncServiceOperationTimeoutDuration() time.Duration {
	return time.Duration(c.AsyncServiceOperationTimeout) * time.Minute
}

func (c *config) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func (c *config) GetAppsDomain() string {
	return c.AppsDomain
}

func (c *config) GetSkipSSLValidation() bool {
	return c.SkipSSLValidation
}

func (c *config) GetArtifactsDirectory() string {
	return c.ArtifactsDirectory
}

func (c *config) GetPersistentAppSpace() string {
	return c.PersistentAppSpace
}
func (c *config) GetPersistentAppOrg() string {
	return c.PersistentAppOrg
}
func (c *config) GetPersistentAppQuotaName() string {
	return c.PersistentAppQuotaName
}

func (c *config) GetNamePrefix() string {
	return c.NamePrefix
}

func (c *config) GetUseExistingUser() bool {
	return c.UseExistingUser
}

func (c *config) GetExistingUser() string {
	return c.ExistingUser
}

func (c *config) GetExistingUserPassword() string {
	return c.ExistingUserPassword
}

func (c *config) GetConfigurableTestPassword() string {
	return c.ConfigurableTestPassword
}

func (c *config) GetShouldKeepUser() bool {
	return c.ShouldKeepUser
}

func (c *config) GetAdminUser() string {
	return c.AdminUser
}

func (c *config) GetAdminPassword() string {
	return c.AdminPassword
}

func (c *config) GetApiEndpoint() string {
	return c.ApiEndpoint
}

func (c *config) GetIncludeSsh() bool {
	return c.IncludeSsh
}

func (c *config) GetIncludeApps() bool {
	return c.IncludeApps
}

func (c *config) GetIncludeBackendCompatiblity() bool {
	return c.IncludeBackendCompatiblity
}

func (c *config) GetIncludeDetect() bool {
	return c.IncludeDetect
}

func (c *config) GetIncludeDocker() bool {
	return c.IncludeDocker
}

func (c *config) GetIncludeInternetDependent() bool {
	return c.IncludeInternetDependent
}

func (c *config) GetIncludeRouteServices() bool {
	return c.IncludeRouteServices
}

func (c *config) GetIncludeRouting() bool {
	return c.IncludeRouting
}

func (c *config) GetIncludeTasks() bool {
	return c.IncludeTasks
}

func (c *config) GetIncludePrivilegedContainerSupport() bool {
	return c.IncludePrivilegedContainerSupport
}

func (c *config) GetIncludeSecurityGroups() bool {
	return c.IncludeSecurityGroups
}

func (c *config) GetIncludeServices() bool {
	return c.IncludeServices
}

func (c *config) GetIncludeSSO() bool {
	return c.IncludeSSO
}

func (c *config) GetIncludeV3() bool {
	return c.IncludeV3
}

func (c *config) GetRubyBuildpackName() string {
	return c.RubyBuildpackName
}

func (c *config) GetGoBuildpackName() string {
	return c.GoBuildpackName
}

func (c *config) GetJavaBuildpackName() string {
	return c.JavaBuildpackName
}

func (c *config) GetNodejsBuildpackName() string {
	return c.NodejsBuildpackName
}

func (c *config) GetBinaryBuildpackName() string {
	return c.BinaryBuildpackName
}

func (c *config) GetPersistentAppHost() string {
	return c.PersistentAppHost
}

func (c *config) GetBackend() string {
	return c.Backend
}
