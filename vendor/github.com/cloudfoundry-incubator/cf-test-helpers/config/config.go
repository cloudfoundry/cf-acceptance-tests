package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

type Config struct {
	ApiEndpoint string `json:"api"`
	AppsDomain  string `json:"apps_domain"`
	UseHttp     bool   `json:"use_http"`

	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`

	UseExistingUser      bool   `json:"use_existing_user"`
	ShouldKeepUser       bool   `json:"keep_user_at_suite_end"`
	ExistingUser         string `json:"existing_user"`
	ExistingUserPassword string `json:"existing_user_password"`

	ConfigurableTestPassword string `json:"test_password"`

	PersistentAppHost      string `json:"persistent_app_host"`
	PersistentAppSpace     string `json:"persistent_app_space"`
	PersistentAppOrg       string `json:"persistent_app_org"`
	PersistentAppQuotaName string `json:"persistent_app_quota_name"`

	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	Backend           string `json:"backend"`

	ArtifactsDirectory string `json:"artifacts_directory"`

	DefaultTimeout               time.Duration `json:"default_timeout"`
	SleepTimeout                 time.Duration `json:"sleep_timeout"`
	DetectTimeout                time.Duration `json:"detect_timeout"`
	CfPushTimeout                time.Duration `json:"cf_push_timeout"`
	LongCurlTimeout              time.Duration `json:"long_curl_timeout"`
	BrokerStartTimeout           time.Duration `json:"broker_start_timeout"`
	AsyncServiceOperationTimeout time.Duration `json:"async_service_operation_timeout"`

	TimeoutScale float64 `json:"timeout_scale"`

	SecureAddress string `json:"secure_address"`

	DockerExecutable      string   `json:"docker_executable"`
	DockerParameters      []string `json:"docker_parameters"`
	DockerRegistryAddress string   `json:"docker_registry_address"`
	DockerPrivateImage    string   `json:"docker_private_image"`
	DockerUser            string   `json:"docker_user"`
	DockerPassword        string   `json:"docker_password"`
	DockerEmail           string   `json:"docker_email"`

	StaticFileBuildpackName string `json:"staticfile_buildpack_name"`
	JavaBuildpackName       string `json:"java_buildpack_name"`
	RubyBuildpackName       string `json:"ruby_buildpack_name"`
	NodejsBuildpackName     string `json:"nodejs_buildpack_name"`
	GoBuildpackName         string `json:"go_buildpack_name"`
	PythonBuildpackName     string `json:"python_buildpack_name"`
	PhpBuildpackName        string `json:"php_buildpack_name"`
	BinaryBuildpackName     string `json:"binary_buildpack_name"`

	IncludeApps                       bool `json:"include_apps"`
	IncludeBackendCompatiblity        bool `json:"include_backend_compatibility"`
	IncludeDetect                     bool `json:"include_detect"`
	IncludeDocker                     bool `json:"include_docker"`
	IncludeInternetDependent          bool `json:"include_internet_dependent"`
	IncludeRouteServices              bool `json:"include_route_services"`
	IncludeRouting                    bool `json:"include_routing"`
	IncludeSecurityGroups             bool `json:"include_security_groups"`
	IncludeServices                   bool `json:"include_services"`
	IncludeSsh                        bool `json:"include_ssh"`
	IncludeV3                         bool `json:"include_v3"`
	IncludeTasks                      bool `json:"include_tasks"`
	IncludePrivilegedContainerSupport bool `json:"include_privileged_container_support"`
	IncludeSSO                        bool `json:"include_sso"`

	NamePrefix string `json:"name_prefix"`
}

var defaults = Config{
	PersistentAppHost:      "CATS-persistent-app",
	PersistentAppSpace:     "CATS-persistent-space",
	PersistentAppOrg:       "CATS-persistent-org",
	PersistentAppQuotaName: "CATS-persistent-quota",

	StaticFileBuildpackName: "staticfile_buildpack",
	JavaBuildpackName:       "java_buildpack",
	RubyBuildpackName:       "ruby_buildpack",
	NodejsBuildpackName:     "nodejs_buildpack",
	GoBuildpackName:         "go_buildpack",
	PythonBuildpackName:     "python_buildpack",
	PhpBuildpackName:        "php_buildpack",
	BinaryBuildpackName:     "binary_buildpack",

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

	ArtifactsDirectory: filepath.Join("..", "results"),

	NamePrefix: "CATS",
}

func (c Config) ScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * c.TimeoutScale)
}

var loadedConfig *Config

func Load(path string, config interface{}) error {
	c, ok := config.(*Config)
	if !ok {
		val := reflect.ValueOf(config).Elem().FieldByName("Config").Addr()
		c = val.Interface().(*Config)
	}

	*c = defaults
	err := loadConfigFromPath(path, config)
	if err != nil {
		return err
	}

	if c.ApiEndpoint == "" {
		return fmt.Errorf("missing configuration 'api'")
	}

	if c.AdminUser == "" {
		return fmt.Errorf("missing configuration 'admin_user'")
	}

	if c.AdminPassword == "" {
		return fmt.Errorf("missing configuration 'admin_password'")
	}

	if c.TimeoutScale <= 0 {
		c.TimeoutScale = 1.0
	}

	return nil
}

func LoadConfig() Config {
	if loadedConfig != nil {
		return *loadedConfig
	}

	var config Config

	err := Load(ConfigPath(), &config)
	if err != nil {
		panic(err)
	}

	loadedConfig = &config
	return config
}

func (c Config) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
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

func ConfigPath() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}
