package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
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

var defaults = Config{
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

var config *Config

func NewConfig() *Config {
	config = &defaults
	err := load(configPath(), config)
	if err != nil {
		panic(err)
	}
	return config
}

func load(path string, config *Config) error {
	err := loadConfigFromPath(path, config)
	if err != nil {
		return err
	}

	if config.ApiEndpoint == "" {
		return fmt.Errorf("missing configuration 'api'")
	}

	if config.AdminUser == "" {
		return fmt.Errorf("missing configuration 'admin_user'")
	}

	if config.AdminPassword == "" {
		return fmt.Errorf("missing configuration 'admin_password'")
	}

	if config.TimeoutScale <= 0 {
		config.TimeoutScale = 1.0
	}

	return nil
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

func (c Config) GetScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * c.TimeoutScale)
}

func (c *Config) DefaultTimeoutDuration() time.Duration {
	return time.Duration(c.DefaultTimeout) * time.Second
}

func (c *Config) LongTimeoutDuration() time.Duration {
	return time.Duration(c.DefaultTimeout) * time.Second
}

func (c *Config) LongCurlTimeoutDuration() time.Duration {
	return time.Duration(c.LongCurlTimeout) * time.Minute
}

func (c *Config) SleepTimeoutDuration() time.Duration {
	return time.Duration(c.SleepTimeout) * time.Second
}

func (c *Config) DetectTimeoutDuration() time.Duration {
	return time.Duration(c.DetectTimeout) * time.Minute
}

func (c *Config) CfPushTimeoutDuration() time.Duration {
	return time.Duration(c.CfPushTimeout) * time.Minute
}

func (c *Config) BrokerStartTimeoutDuration() time.Duration {
	return time.Duration(c.BrokerStartTimeout) * time.Minute
}

func (c *Config) AsyncServiceOperationTimeoutDuration() time.Duration {
	return time.Duration(c.AsyncServiceOperationTimeout) * time.Minute
}

func (c Config) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func (c *Config) GetAppsDomain() string {
	return c.AppsDomain
}

func (c *Config) GetSkipSSLValidation() bool {
	return c.SkipSSLValidation
}

func (c *Config) GetArtifactsDirectory() string {
	return c.ArtifactsDirectory
}

func (c *Config) GetPersistentAppSpace() string {
	return c.PersistentAppSpace
}
func (c *Config) GetPersistentAppOrg() string {
	return c.PersistentAppOrg
}
func (c *Config) GetPersistentAppQuotaName() string {
	return c.PersistentAppQuotaName
}

func (c *Config) GetNamePrefix() string {
	return c.NamePrefix
}

func (c *Config) GetUseExistingUser() bool {
	return c.UseExistingUser
}

func (c *Config) GetExistingUser() string {
	return c.ExistingUser
}

func (c *Config) GetExistingUserPassword() string {
	return c.ExistingUserPassword
}

func (c *Config) GetConfigurableTestPassword() string {
	return c.ConfigurableTestPassword
}

func (c *Config) GetShouldKeepUser() bool {
	return c.ShouldKeepUser
}

func (c *Config) GetAdminUser() string {
	return c.AdminUser
}

func (c *Config) GetAdminPassword() string {
	return c.AdminPassword
}

func (c *Config) GetApiEndpoint() string {
	return c.ApiEndpoint
}

func (c *Config) GetIncludeSsh() bool {
	return c.IncludeSsh
}

func (c *Config) GetIncludeApps() bool {
	return c.IncludeApps
}

func (c *Config) GetIncludeBackendCompatiblity() bool {
	return c.IncludeBackendCompatiblity
}

func (c *Config) GetIncludeDetect() bool {
	return c.IncludeDetect
}

func (c *Config) GetIncludeDocker() bool {
	return c.IncludeDocker
}

func (c *Config) GetIncludeInternetDependent() bool {
	return c.IncludeInternetDependent
}

func (c *Config) GetIncludeRouteServices() bool {
	return c.IncludeRouteServices
}

func (c *Config) GetIncludeRouting() bool {
	return c.IncludeRouting
}

func (c *Config) GetIncludeTasks() bool {
	return c.IncludeTasks
}

func (c *Config) GetIncludePrivilegedContainerSupport() bool {
	return c.IncludePrivilegedContainerSupport
}

func (c *Config) GetIncludeSecurityGroups() bool {
	return c.IncludeSecurityGroups
}

func (c *Config) GetIncludeServices() bool {
	return c.IncludeServices
}

func (c *Config) GetIncludeSSO() bool {
	return c.IncludeSSO
}

func (c *Config) GetIncludeV3() bool {
	return c.IncludeV3
}

func (c *Config) GetRubyBuildpackName() string {
	return c.RubyBuildpackName
}

func (c *Config) GetGoBuildpackName() string {
	return c.GoBuildpackName
}

func (c *Config) GetJavaBuildpackName() string {
	return c.JavaBuildpackName
}

func (c *Config) GetNodejsBuildpackName() string {
	return c.NodejsBuildpackName
}

func (c *Config) GetBinaryBuildpackName() string {
	return c.BinaryBuildpackName
}

func (c *Config) GetPersistentAppHost() string {
	return c.PersistentAppHost
}

func (c *Config) GetBackend() string {
	return c.Backend
}
