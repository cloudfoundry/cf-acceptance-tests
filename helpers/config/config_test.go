package config_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/validationerrors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type requiredConfig struct {
	// required
	ApiEndpoint       *string `json:"api"`
	AdminUser         *string `json:"admin_user"`
	AdminPassword     *string `json:"admin_password"`
	SkipSSLValidation *bool   `json:"skip_ssl_validation"`
	AppsDomain        *string `json:"apps_domain"`
	UseHttp           *bool   `json:"use_http"`
}

type testConfig struct {
	// required
	ApiEndpoint       *string `json:"api"`
	AdminUser         *string `json:"admin_user"`
	AdminPassword     *string `json:"admin_password"`
	SkipSSLValidation *bool   `json:"skip_ssl_validation"`
	AppsDomain        *string `json:"apps_domain"`
	UseHttp           *bool   `json:"use_http,omitempty"`

	// timeouts
	DefaultTimeout               *int `json:"default_timeout,omitempty"`
	CfPushTimeout                *int `json:"cf_push_timeout,omitempty"`
	LongCurlTimeout              *int `json:"long_curl_timeout,omitempty"`
	BrokerStartTimeout           *int `json:"broker_start_timeout,omitempty"`
	AsyncServiceOperationTimeout *int `json:"async_service_operation_timeout,omitempty"`
	DetectTimeout                *int `json:"detect_timeout,omitempty"`
	SleepTimeout                 *int `json:"sleep_timeout,omitempty"`

	TimeoutScale *float64 `json:"timeout_scale,omitempty"`

	// optional
	Backend                       *string `json:"backend,omitempty"`
	IncludePrivateDockerRegistry  *bool   `json:"include_private_docker_registry,omitempty"`
	PrivateDockerRegistryImage    *string `json:"private_docker_registry_image,omitempty"`
	PrivateDockerRegistryUsername *string `json:"private_docker_registry_username,omitempty"`
	PrivateDockerRegistryPassword *string `json:"private_docker_registry_password,omitempty"`
	PublicDockerAppImage          *string `json:"public_docker_app_image,omitempty"`

	IncludeIsolationSegments        *bool   `json:"include_isolation_segments,omitempty"`
	IncludeRoutingIsolationSegments *bool   `json:"include_routing_isolation_segments,omitempty"`
	IsolationSegmentName            *string `json:"isolation_segment_name,omitempty"`
	IsolationSegmentDomain          *string `json:"isolation_segment_domain,omitempty"`
	UnallocatedIPForSecurityGroup   *string `json:"unallocated_ip_for_security_group"`
}

type allConfig struct {
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

	IsolationSegmentName   *string `json:"isolation_segment_name"`
	IsolationSegmentDomain *string `json:"isolation_segment_domain"`

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

	IncludeApps                       *bool `json:"include_apps"`
	IncludeBackendCompatiblity        *bool `json:"include_backend_compatibility"`
	IncludeCapiExperimental           *bool `json:"include_capi_experimental"`
	IncludeCapiNoBridge               *bool `json:"include_capi_no_bridge"`
	IncludeContainerNetworking        *bool `json:"include_container_networking"`
	IncludeCredHub                    *bool `json:"include_credhub"`
	IncludeDetect                     *bool `json:"include_detect"`
	IncludeDocker                     *bool `json:"include_docker"`
	IncludeInternetDependent          *bool `json:"include_internet_dependent"`
	IncludePrivateDockerRegistry      *bool `json:"include_private_docker_registry"`
	IncludePersistentApp              *bool `json:"include_persistent_app"`
	IncludePrivilegedContainerSupport *bool `json:"include_privileged_container_support"`
	IncludeRouteServices              *bool `json:"include_route_services"`
	IncludeRouting                    *bool `json:"include_routing"`
	IncludeSSO                        *bool `json:"include_sso"`
	IncludeSecurityGroups             *bool `json:"include_security_groups"`
	IncludeServices                   *bool `json:"include_services"`
	IncludeServiceInstanceSharing     *bool `json:"include_service_instance_sharing"`
	IncludeSsh                        *bool `json:"include_ssh"`
	IncludeTasks                      *bool `json:"include_tasks"`
	IncludeV3                         *bool `json:"include_v3"`
	IncludeZipkin                     *bool `json:"include_zipkin"`
	IncludeIsolationSegments          *bool `json:"include_isolation_segments"`

	PrivateDockerRegistryImage    *string `json:"private_docker_registry_image"`
	PrivateDockerRegistryUsername *string `json:"private_docker_registry_username"`
	PrivateDockerRegistryPassword *string `json:"private_docker_registry_password"`
	PublicDockerAppImage          *string `json:"public_docker_app_image"`

	NamePrefix *string `json:"name_prefix"`
}

var tmpFilePath string
var err error
var errors Errors
var requiredCfg requiredConfig
var testCfg testConfig
var originalConfig string

func writeConfigFile(updatedConfig interface{}) string {
	configFile, err := ioutil.TempFile("", "cf-test-helpers-config")
	Expect(err).NotTo(HaveOccurred())

	encoder := json.NewEncoder(configFile)
	err = encoder.Encode(updatedConfig)

	Expect(err).NotTo(HaveOccurred())

	err = configFile.Close()
	Expect(err).NotTo(HaveOccurred())

	return configFile.Name()
}

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

var _ = Describe("Config", func() {
	BeforeEach(func() {
		testCfg = testConfig{}
		testCfg.ApiEndpoint = ptrToString("api.bosh-lite.com")
		testCfg.AdminUser = ptrToString("admin")
		testCfg.AdminPassword = ptrToString("admin")
		testCfg.SkipSSLValidation = ptrToBool(true)
		testCfg.AppsDomain = ptrToString("cf-app.bosh-lite.com")
	})

	JustBeforeEach(func() {
		tmpFilePath = writeConfigFile(&testCfg)
	})

	AfterEach(func() {
		err := os.Remove(tmpFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should have the right defaults", func() {
		requiredCfg := requiredConfig{}
		requiredCfg.ApiEndpoint = testCfg.ApiEndpoint
		requiredCfg.AdminUser = testCfg.AdminUser
		requiredCfg.AdminPassword = testCfg.AdminPassword
		requiredCfg.SkipSSLValidation = testCfg.SkipSSLValidation
		requiredCfg.AppsDomain = testCfg.AppsDomain
		requiredCfg.UseHttp = ptrToBool(true)

		requiredCfgFilePath := writeConfigFile(requiredCfg)
		config, err := cfg.NewCatsConfig(requiredCfgFilePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(config.GetIncludeApps()).To(BeTrue())
		Expect(config.GetIncludePersistentApp()).To(BeTrue())
		Expect(config.GetPersistentAppHost()).To(Equal("CATS-persistent-app"))

		Expect(config.GetPersistentAppOrg()).To(Equal("CATS-persistent-org"))
		Expect(config.GetPersistentAppQuotaName()).To(Equal("CATS-persistent-quota"))
		Expect(config.GetPersistentAppSpace()).To(Equal("CATS-persistent-space"))

		Expect(config.GetIsolationSegmentName()).To(Equal(""))
		Expect(config.GetIsolationSegmentDomain()).To(Equal(""))

		Expect(config.GetIncludeApps()).To(BeTrue())
		Expect(config.GetIncludeDetect()).To(BeTrue())
		Expect(config.GetIncludeRouting()).To(BeTrue())
		Expect(config.GetIncludeV3()).To(BeTrue())

		Expect(config.GetIncludeBackendCompatiblity()).To(BeFalse())
		Expect(config.GetIncludeCapiExperimental()).To(BeFalse())
		Expect(config.GetIncludeCapiNoBridge()).To(BeFalse())
		Expect(config.GetIncludeDocker()).To(BeFalse())
		Expect(config.GetIncludeInternetDependent()).To(BeFalse())
		Expect(config.GetIncludeRouteServices()).To(BeFalse())
		Expect(config.GetIncludeContainerNetworking()).To(BeFalse())
		Expect(config.GetIncludeSecurityGroups()).To(BeFalse())
		Expect(config.GetIncludeServices()).To(BeFalse())
		Expect(config.GetIncludeSsh()).To(BeFalse())
		Expect(config.GetIncludeIsolationSegments()).To(BeFalse())
		Expect(config.GetIncludePrivateDockerRegistry()).To(BeFalse())
		Expect(config.GetIncludePrivilegedContainerSupport()).To(BeFalse())
		Expect(config.GetIncludeZipkin()).To(BeFalse())
		Expect(config.GetIncludeSSO()).To(BeFalse())
		Expect(config.GetIncludeTasks()).To(BeFalse())
		Expect(config.GetIncludeCredhubAssisted()).To(BeFalse())
		Expect(config.GetIncludeCredhubNonAssisted()).To(BeFalse())
		Expect(config.GetIncludeServiceInstanceSharing()).To(BeFalse())

		Expect(config.GetBackend()).To(Equal(""))

		Expect(config.GetUseExistingUser()).To(Equal(false))
		Expect(config.GetConfigurableTestPassword()).To(Equal(""))
		Expect(config.GetShouldKeepUser()).To(Equal(false))

		Expect(config.GetExistingOrganization()).To(Equal(""))
		Expect(config.GetUseExistingOrganization()).To(Equal(false))

		Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(4 * time.Minute))
		Expect(config.BrokerStartTimeoutDuration()).To(Equal(10 * time.Minute))
		Expect(config.CfPushTimeoutDuration()).To(Equal(4 * time.Minute))
		Expect(config.DefaultTimeoutDuration()).To(Equal(60 * time.Second))
		Expect(config.LongCurlTimeoutDuration()).To(Equal(4 * time.Minute))

		Expect(config.GetScaledTimeout(1)).To(Equal(time.Duration(2)))

		Expect(config.GetArtifactsDirectory()).To(Equal(filepath.Join("..", "results")))

		Expect(config.GetPrivateDockerRegistryImage()).To(Equal(""))
		Expect(config.GetPrivateDockerRegistryUsername()).To(Equal(""))
		Expect(config.GetPrivateDockerRegistryPassword()).To(Equal(""))

		Expect(config.GetNamePrefix()).To(Equal("CATS"))

		Expect(config.Protocol()).To(Equal("http://"))

		// undocumented
		Expect(config.DetectTimeoutDuration()).To(Equal(10 * time.Minute))
		Expect(config.SleepTimeoutDuration()).To(Equal(60 * time.Second))

		Expect(config.GetPublicDockerAppImage()).To(Equal("cloudfoundry/diego-docker-app-custom:latest"))
		Expect(config.GetUnallocatedIPForSecurityGroup()).To(Equal("10.0.244.255"))
	})

	Context("when all values are null", func() {
		It("returns an error", func() {
			allCfgFilePath := writeConfigFile(&allConfig{})
			_, err := cfg.NewCatsConfig(allCfgFilePath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'api' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'apps_domain' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'use_http' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'admin_password' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'admin_user' must not be null"))

			// Expect(err.Error()).To(ContainSubstring("'existing_user' must not be null"))
			// Expect(err.Error()).To(ContainSubstring("'existing_user_password' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'keep_user_at_suite_end' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'use_existing_user' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'test_password' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'persistent_app_host' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'persistent_app_org' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'persistent_app_quota_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'persistent_app_space' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'isolation_segment_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'isolation_segment_domain' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'backend' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'skip_ssl_validation' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'artifacts_directory' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'async_service_operation_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'broker_start_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'cf_push_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'default_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'detect_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'long_curl_timeout' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'sleep_timeout' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'timeout_scale' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'binary_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'go_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'java_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'nodejs_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'php_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'python_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'ruby_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'staticfile_buildpack_name' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'include_apps' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_backend_compatibility' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'backend' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_capi_experimental' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_capi_no_bridge' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_detect' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_docker' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_internet_dependent' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_persistent_app' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_private_docker_registry' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_privileged_container_support' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_route_services' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_routing' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_container_networking' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_sso' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_security_groups' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_services' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_service_instance_sharing' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_ssh' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_tasks' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_v3' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_zipkin' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_isolation_segments' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'private_docker_registry_image' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'private_docker_registry_username' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'private_docker_registry_password' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'name_prefix' must not be null"))
		})
	})

	Context("when values with default are overriden", func() {
		BeforeEach(func() {
			testCfg.DefaultTimeout = ptrToInt(12)
			testCfg.CfPushTimeout = ptrToInt(34)
			testCfg.LongCurlTimeout = ptrToInt(56)
			testCfg.BrokerStartTimeout = ptrToInt(78)
			testCfg.AsyncServiceOperationTimeout = ptrToInt(90)
			testCfg.DetectTimeout = ptrToInt(100)
			testCfg.SleepTimeout = ptrToInt(101)
			testCfg.TimeoutScale = ptrToFloat(1.0)
			testCfg.UnallocatedIPForSecurityGroup = ptrToString("192.168.0.1")
		})

		It("respects the overriden values", func() {
			config, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(config.DefaultTimeoutDuration()).To(Equal(12 * time.Second))
			Expect(config.CfPushTimeoutDuration()).To(Equal(34 * time.Second))
			Expect(config.LongCurlTimeoutDuration()).To(Equal(56 * time.Second))
			Expect(config.BrokerStartTimeoutDuration()).To(Equal(78 * time.Second))
			Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(90 * time.Second))
			Expect(config.DetectTimeoutDuration()).To(Equal(100 * time.Second))
			Expect(config.SleepTimeoutDuration()).To(Equal(101 * time.Second))
			Expect(config.SleepTimeoutDuration()).To(Equal(101 * time.Second))
			Expect(config.GetUnallocatedIPForSecurityGroup()).To(Equal("192.168.0.1"))
		})
	})

	Context("when including private docker registry tests", func() {
		BeforeEach(func() {
			testCfg.IncludePrivateDockerRegistry = ptrToBool(true)
			testCfg.PrivateDockerRegistryImage = ptrToString("value")
			testCfg.PrivateDockerRegistryUsername = ptrToString("value")
			testCfg.PrivateDockerRegistryPassword = ptrToString("value")
		})

		Context("when image is an empty string", func() {
			BeforeEach(func() {
				testCfg.PrivateDockerRegistryImage = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'private_docker_registry_image' must be provided if 'include_private_docker_registry' is true"))
			})
		})

		Context("when username is an empty string", func() {
			BeforeEach(func() {
				testCfg.PrivateDockerRegistryUsername = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'private_docker_registry_username' must be provided if 'include_private_docker_registry' is true"))
			})
		})

		Context("when password is an empty string", func() {
			BeforeEach(func() {
				testCfg.PrivateDockerRegistryPassword = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'private_docker_registry_password' must be provided if 'include_private_docker_registry' is true"))
			})
		})
	})

	Context("when including public_docker_app_image", func() {
		Context("when image name is set", func() {
			var image = "some-image"
			BeforeEach(func() {
				testCfg.PublicDockerAppImage = ptrToString(image)
			})

			It("has the value in the config", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.GetPublicDockerAppImage()).To(Equal(image))
			})
		})

		Context("when image is an empty string", func() {
			BeforeEach(func() {
				testCfg.PublicDockerAppImage = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'public_docker_app_image' must be set to a valid image source"))
			})
		})
	})

	Context("when including isolation segment tests", func() {
		BeforeEach(func() {
			testCfg.IncludeIsolationSegments = ptrToBool(true)
			testCfg.IsolationSegmentName = ptrToString("value")
		})

		Context("when image is an empty string", func() {
			BeforeEach(func() {
				testCfg.IsolationSegmentName = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'isolation_segment_name' must be provided if 'include_isolation_segments' is true"))
			})
		})
	})

	Context("when including routing isolation segment tests", func() {
		BeforeEach(func() {
			testCfg.IncludeRoutingIsolationSegments = ptrToBool(true)
			testCfg.IsolationSegmentName = ptrToString("value")
			testCfg.IsolationSegmentDomain = ptrToString("value")
		})

		Context("when isolation_segment_name is an empty string", func() {
			BeforeEach(func() {
				testCfg.IsolationSegmentName = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'isolation_segment_name' must be provided if 'include_routing_isolation_segments' is true"))
			})
		})

		Context("when isolation_segment_domain is an empty string", func() {
			BeforeEach(func() {
				testCfg.IsolationSegmentDomain = ptrToString("")
			})

			It("returns an error", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(config).To(BeNil())
				Expect(err).To(MatchError("* Invalid configuration: 'isolation_segment_domain' must be provided if 'include_routing_isolation_segments' is true"))
			})
		})
	})

	Describe("error aggregation", func() {
		BeforeEach(func() {
			testCfg.AdminPassword = nil
			testCfg.ApiEndpoint = ptrToString("invalid-url.asdf")
		})

		It("aggregates all errors", func() {
			config, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(config).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("* 'admin_password' must not be null"))
			Expect(err.Error()).To(ContainSubstring("* Invalid configuration for 'api' <invalid-url.asdf>"))
		})
	})

	Describe(`GetBackend`, func() {
		Context("when the backend is set to `dea`", func() {
			BeforeEach(func() {
				testCfg.Backend = ptrToString("dea")
			})

			It("returns `dea`", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetBackend()).To(Equal("dea"))
			})
		})

		Context("when the backend is set to `diego`", func() {
			BeforeEach(func() {
				testCfg.Backend = ptrToString("diego")
			})

			It("returns `diego`", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetBackend()).To(Equal("diego"))
			})
		})

		Context("when the backend is empty", func() {
			BeforeEach(func() {
				testCfg.Backend = ptrToString("")
			})

			It("returns an empty string", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetBackend()).To(Equal(""))
			})
		})

		Context("when the backend is set to any other value", func() {
			BeforeEach(func() {
				testCfg.Backend = ptrToString("asdfasdf")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'backend' must be 'diego', 'dea', or empty but was set to 'asdfasdf'"))
			})
		})
	})

	Describe("GetApiEndpoint", func() {
		It(`returns the URL`, func() {
			cfg, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetApiEndpoint()).To(Equal("api.bosh-lite.com"))
		})

		Context("when url is an IP address", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("10.244.0.34") // api.bosh-lite.com
			})

			It("returns the IP address", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetApiEndpoint()).To(Equal("10.244.0.34"))
			})
		})

		Context("when the domain does not resolve", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("some-url-that-does-not-resolve.com.some-url-that-does-not-resolve.com")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such host"))
			})
		})

		Context("when the url is empty", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank"))
			})
		})

		Context("when the url is invalid", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("_bogus%%%")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'api' must be a valid URL but was set to '_bogus%%%'"))
			})
		})

		Context("when the ApiEndpoint is nil", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'api' must not be null"))
			})
		})
	})

	Describe("GetAppsDomain", func() {
		It("returns the domain", func() {
			c, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.GetAppsDomain()).To(Equal("cf-app.bosh-lite.com"))
		})

		Context("when the domain is not valid", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = ptrToString("_bogus%%%")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'apps_domain' must be a valid URL but was set to '_bogus%%%'"))
			})
		})

		Context("when the AppsDomain is an IP address (which is invalid for AppsDomain)", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = ptrToString("10.244.0.34")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such host"))
			})
		})

		Context("when the AppsDomain is nil", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'apps_domain' must not be null"))
			})
		})
	})

	Describe("GetAdminUser", func() {
		It("returns the admin user", func() {
			c, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.GetAdminUser()).To(Equal("admin"))
		})

		Context("when the admin user is blank", func() {
			BeforeEach(func() {
				*testCfg.AdminUser = ""
			})
			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_user' must be provided"))
			})
		})

		Context("when the admin user is nil", func() {
			BeforeEach(func() {
				testCfg.AdminUser = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_user' must not be null"))
			})
		})
	})

	Describe("GetAdminPassword", func() {
		It("returns the admin password", func() {
			c, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.GetAdminPassword()).To(Equal("admin"))
		})

		Context("when the admin user password is blank", func() {
			BeforeEach(func() {
				testCfg.AdminPassword = ptrToString("")
			})
			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_password' must be provided"))
			})
		})

		Context("when the admin user password is nil", func() {
			BeforeEach(func() {
				testCfg.AdminPassword = nil
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_password' must not be null"))
			})
		})
	})
})
