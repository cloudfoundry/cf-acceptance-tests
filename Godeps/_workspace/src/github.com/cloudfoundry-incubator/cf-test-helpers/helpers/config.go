package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
)

type Config struct {
	ApiEndpoint  string `json:"api"`
	SystemDomain string `json:"system_domain"`
	ClientSecret string `json:"client_secret"`
	AppsDomain   string `json:"apps_domain"`
	UseHttp      bool   `json:"use_http"`

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

	SkipSSLValidation    bool   `json:"skip_ssl_validation"`
	Backend              string `json:"backend"`
	IncludeRouteServices bool   `json:"include_route_services"`
	IncludeDiegoDocker   bool   `json:"include_diego_docker"`
	IncludeTasks         bool   `json:"include_tasks"`

	ArtifactsDirectory string `json:"artifacts_directory"`

	DefaultTimeout     time.Duration `json:"default_timeout"`
	DetectTimeout      time.Duration `json:"detect_timeout"`
	CfPushTimeout      time.Duration `json:"cf_push_timeout"`
	LongCurlTimeout    time.Duration `json:"long_curl_timeout"`
	BrokerStartTimeout time.Duration `json:"broker_start_timeout"`

	TimeoutScale float64 `json:"timeout_scale"`

	SyslogDrainPort int    `json:"syslog_drain_port"`
	SyslogIpAddress string `json:"syslog_ip_address"`

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
}

func (c Config) ScaledTimeout(timeout time.Duration) time.Duration {
	return time.Duration(float64(timeout) * c.TimeoutScale)
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.ApiEndpoint == "" {
		panic("missing configuration 'api'")
	}

	if loadedConfig.AdminUser == "" {
		panic("missing configuration 'admin_user'")
	}

	if loadedConfig.AdminPassword == "" {
		panic("missing configuration 'admin_password'")
	}

	if loadedConfig.TimeoutScale <= 0 {
		loadedConfig.TimeoutScale = 1.0
	}

	runner.SkipSSLValidation = loadedConfig.SkipSSLValidation

	return *loadedConfig
}

func (c Config) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func loadConfigJsonFromPath() *Config {
	var config *Config = &Config{
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

		ArtifactsDirectory: filepath.Join("..", "results"),
	}

	path := configPath()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return config
}

func configPath() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}
