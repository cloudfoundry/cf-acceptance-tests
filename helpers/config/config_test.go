package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"

	. "github.com/onsi/ginkgo/v2"
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
	requiredConfig

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
	PrivateDockerRegistryImage    *string `json:"private_docker_registry_image,omitempty"`
	PrivateDockerRegistryUsername *string `json:"private_docker_registry_username,omitempty"`
	PrivateDockerRegistryPassword *string `json:"private_docker_registry_password,omitempty"`
	PublicDockerAppImage          *string `json:"public_docker_app_image,omitempty"`
	CatnipDockerAppImage          *string `json:"catnip_docker_app_image,omitempty"`

	IsolationSegmentName   *string `json:"isolation_segment_name,omitempty"`
	IsolationSegmentDomain *string `json:"isolation_segment_domain,omitempty"`

	UnallocatedIPForSecurityGroup *string `json:"unallocated_ip_for_security_group"`

	UseWindowsTestTask    *bool   `json:"use_windows_test_task,omitempty"`
	UseWindowsContextPath *bool   `json:"use_windows_context_path,omitempty"`
	WindowsStack          *string `json:"windows_stack,omitempty"`

	ReporterConfig *testReporterConfig `json:"reporter_config"`

	Stacks *[]string `json:"stacks,omitempty"`

	VolumeServiceName     *string `json:"volume_service_name,omitempty"`
	VolumeServicePlanName *string `json:"volume_service_plan_name,omitempty"`

	IncludeAppSyslogTcp             *bool `json:"include_app_syslog_tcp,omitempty"`
	IncludeApps                     *bool `json:"include_apps,omitempty"`
	IncludeContainerNetworking      *bool `json:"include_container_networking,omitempty"`
	IncludeDeployments              *bool `json:"include_deployments,omitempty"`
	IncludeDetect                   *bool `json:"include_detect,omitempty"`
	IncludeDocker                   *bool `json:"include_docker,omitempty"`
	IncludeFileBasedServiceBindings *bool `json:"include_file_based_service_bindings,omitempty"`
	IncludeInternetDependent        *bool `json:"include_internet_dependent,omitempty"`
	IncludeIsolationSegments        *bool `json:"include_isolation_segments,omitempty"`
	IncludePrivateDockerRegistry    *bool `json:"include_private_docker_registry,omitempty"`
	IncludeRouteServices            *bool `json:"include_route_services,omitempty"`
	IncludeRouting                  *bool `json:"include_routing,omitempty"`
	IncludeRoutingIsolationSegments *bool `json:"include_routing_isolation_segments,omitempty"`
	IncludeSSO                      *bool `json:"include_sso,omitempty"`
	IncludeSecurityGroups           *bool `json:"include_security_groups,omitempty"`
	IncludeServiceDiscovery         *bool `json:"include_service_discovery,omitempty"`
	IncludeServiceInstanceSharing   *bool `json:"include_service_instance_sharing,omitempty"`
	IncludeServices                 *bool `json:"include_services,omitempty"`
	IncludeUserProvidedServices     *bool `json:"include_user_provided_services,omitempty"`
	IncludeSsh                      *bool `json:"include_ssh,omitempty"`
	IncludeTCPIsolationSegments     *bool `json:"include_tcp_isolation_segments,omitempty"`
	IncludeHTTP2Routing             *bool `json:"include_http2_routing,omitempty"`
	IncludeTCPRouting               *bool `json:"include_tcp_routing,omitempty"`
	IncludeTasks                    *bool `json:"include_tasks,omitempty"`
	IncludeV3                       *bool `json:"include_v3,omitempty"`
	IncludeVolumeServices           *bool `json:"include_volume_services,omitempty"`
	IncludeZipkin                   *bool `json:"include_zipkin,omitempty"`
	IncludeWindows                  *bool `json:"include_windows,omitempty"`
	IncludeIPv6                     *bool `json:"include_ipv6,omitempty"`

	BinaryBuildpackName     *string `json:"binary_buildpack_name,omitempty"`
	GoBuildpackName         *string `json:"go_buildpack_name,omitempty"`
	HwcBuildpackName        *string `json:"hwc_buildpack_name,omitempty"`
	JavaBuildpackName       *string `json:"java_buildpack_name,omitempty"`
	NginxBuildpackName      *string `json:"nginx_buildpack_name,omitempty"`
	NodejsBuildpackName     *string `json:"nodejs_buildpack_name,omitempty"`
	RBuildpackName          *string `json:"r_buildpack_name,omitempty"`
	RubyBuildpackName       *string `json:"ruby_buildpack_name,omitempty"`
	StaticFileBuildpackName *string `json:"staticfile_buildpack_name,omitempty"`
	PythonBuildpackName     *string `json:"python_buildpack_name,omitempty"`
}

type nullConfig struct {
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

	IsolationSegmentName   *string `json:"isolation_segment_name"`
	IsolationSegmentDomain *string `json:"isolation_segment_domain"`

	SkipSSLValidation *bool `json:"skip_ssl_validation"`

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
	HwcBuildpackName        *string `json:"hwc_buildpack_name"`
	JavaBuildpackName       *string `json:"java_buildpack_name"`
	NginxBuildpackName      *string `json:"nginx_buildpack_name"`
	NodejsBuildpackName     *string `json:"nodejs_buildpack_name"`
	RBuildpackName          *string `json:"r_buildpack_name"`
	RubyBuildpackName       *string `json:"ruby_buildpack_name"`
	StaticFileBuildpackName *string `json:"staticfile_buildpack_name"`
	PythonBuildpackName     *string `json:"python_buildpack_name"`

	ReporterConfig *testReporterConfig `json:"reporter_config"`

	IncludeApps                     *bool `json:"include_apps"`
	IncludeContainerNetworking      *bool `json:"include_container_networking"`
	IncludeDetect                   *bool `json:"include_detect"`
	IncludeDocker                   *bool `json:"include_docker"`
	IncludeFileBasedServiceBindings *bool `json:"include_file_based_service_bindings"`
	IncludeInternetDependent        *bool `json:"include_internet_dependent"`
	IncludePrivateDockerRegistry    *bool `json:"include_private_docker_registry"`
	IncludeRouteServices            *bool `json:"include_route_services"`
	IncludeRouting                  *bool `json:"include_routing"`
	IncludeSSO                      *bool `json:"include_sso"`
	IncludeSecurityGroups           *bool `json:"include_security_groups"`
	IncludeServices                 *bool `json:"include_services"`
	IncludeUserProvidedServices     *bool `json:"include_user_provided_services"`
	IncludeServiceInstanceSharing   *bool `json:"include_service_instance_sharing"`
	IncludeSsh                      *bool `json:"include_ssh"`
	IncludeTasks                    *bool `json:"include_tasks"`
	IncludeV3                       *bool `json:"include_v3"`
	IncludeWindows                  *bool `json:"include_windows"`
	IncludeZipkin                   *bool `json:"include_zipkin"`
	IncludeIsolationSegments        *bool `json:"include_isolation_segments"`
	IncludeRoutingIsolationSegments *bool `json:"include_routing_isolation_segments"`
	IncludeHTTP2Routing             *bool `json:"include_http2_routing"`
	IncludeTCPRouting               *bool `json:"include_tcp_routing"`
	IncludeServiceDiscovery         *bool `json:"include_service_discovery"`
	IncludeVolumeServices           *bool `json:"include_volume_services"`
	IncludeTCPIsolationSegments     *bool `json:"include_tcp_isolation_segments"`
	IncludeAppSyslogTcp             *bool `json:"include_app_syslog_tcp"`

	CredhubMode         *string `json:"credhub_mode"`
	CredhubLocation     *string `json:"credhub_location"`
	CredhubClientName   *string `json:"credhub_client"`
	CredhubClientSecret *string `json:"credhub_secret"`

	PrivateDockerRegistryImage    *string `json:"private_docker_registry_image"`
	PrivateDockerRegistryUsername *string `json:"private_docker_registry_username"`
	PrivateDockerRegistryPassword *string `json:"private_docker_registry_password"`
	PublicDockerAppImage          *string `json:"public_docker_app_image"`

	NamePrefix *string `json:"name_prefix"`

	Stacks *[]string `json:"stacks"`

	UnallocatedIPForSecurityGroup *string `json:"unallocated_ip_for_security_group"`
	UseWindowsContextPath         *bool   `json:"use_windows_context_path"`
	UseWindowsTestTask            *bool   `json:"use_windows_test_task"`
}

type testReporterConfig struct {
	HoneyCombWriteKey string                 `json:"honeycomb_write_key"`
	HoneyCombDataset  string                 `json:"honeycomb_dataset"`
	CustomTags        map[string]interface{} `json:"custom_tags"`
}

const BoshLiteDomain = "bosh-lite.env.wg-ard.ci.cloudfoundry.org"

var tmpFilePath string
var testCfg testConfig

func writeConfigFile(updatedConfig interface{}) string {
	configFile, err := os.CreateTemp("", "cf-test-helpers-config")
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
		testCfg.ApiEndpoint = ptrToString("api." + BoshLiteDomain)
		testCfg.AdminUser = ptrToString("admin")
		testCfg.AdminPassword = ptrToString("admin")
		testCfg.SkipSSLValidation = ptrToBool(true)
		testCfg.AppsDomain = ptrToString("cf-app." + BoshLiteDomain)
		testCfg.UseHttp = ptrToBool(false)
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

		Expect(config.GetIsolationSegmentName()).To(Equal(""))
		Expect(config.GetIsolationSegmentDomain()).To(Equal(""))

		Expect(config.GetIncludeAppSyslogTcp()).To(BeTrue())
		Expect(config.GetIncludeApps()).To(BeTrue())
		Expect(config.GetIncludeDeployments()).To(BeFalse())
		Expect(config.GetIncludeDetect()).To(BeTrue())
		Expect(config.GetIncludeRouting()).To(BeTrue())
		Expect(config.GetIncludeV3()).To(BeTrue())

		Expect(config.GetIncludeDocker()).To(BeFalse())
		Expect(config.GetIncludeFileBasedServiceBindings()).To(BeFalse())
		Expect(config.GetIncludeIPv6()).To(BeFalse())
		Expect(config.GetIncludeInternetDependent()).To(BeFalse())
		Expect(config.GetIncludeRouteServices()).To(BeFalse())
		Expect(config.GetIncludeContainerNetworking()).To(BeFalse())
		Expect(config.GetIncludeSecurityGroups()).To(BeFalse())
		Expect(config.GetIncludeServiceDiscovery()).To(BeFalse())
		Expect(config.GetIncludeServices()).To(BeFalse())
		Expect(config.GetIncludeUserProvidedServices()).To(BeFalse())
		Expect(config.GetIncludeSsh()).To(BeFalse())
		Expect(config.GetIncludeIsolationSegments()).To(BeFalse())
		Expect(config.GetIncludeRoutingIsolationSegments()).To(BeFalse())
		Expect(config.GetIncludeTCPIsolationSegments()).To(BeFalse())
		Expect(config.GetIncludePrivateDockerRegistry()).To(BeFalse())
		Expect(config.GetIncludeZipkin()).To(BeFalse())
		Expect(config.GetIncludeSSO()).To(BeFalse())
		Expect(config.GetIncludeTasks()).To(BeFalse())
		Expect(config.GetIncludeCredhubAssisted()).To(BeFalse())
		Expect(config.GetIncludeCredhubNonAssisted()).To(BeFalse())
		Expect(config.GetIncludeServiceInstanceSharing()).To(BeFalse())
		Expect(config.GetIncludeHTTP2Routing()).To(BeFalse())
		Expect(config.GetIncludeTCPRouting()).To(BeFalse())
		Expect(config.GetIncludeVolumeServices()).To(BeFalse())

		Expect(config.GetIncludeWindows()).To(BeFalse())
		Expect(config.GetUseWindowsTestTask()).To(BeFalse())
		Expect(config.GetUseWindowsContextPath()).To(BeFalse())
		Expect(config.GetWindowsStack()).To(Equal("windows"))

		Expect(config.GetIncludeServiceDiscovery()).To(BeFalse())

		testReporterConfig := config.GetReporterConfig()
		Expect(testReporterConfig.HoneyCombDataset).To(Equal(""))
		Expect(testReporterConfig.HoneyCombWriteKey).To(Equal(""))

		Expect(config.GetUseExistingUser()).To(Equal(false))
		Expect(config.GetConfigurableTestPassword()).To(Equal(""))
		Expect(config.GetShouldKeepUser()).To(Equal(false))

		Expect(config.GetExistingOrganization()).To(Equal(""))
		Expect(config.GetUseExistingOrganization()).To(Equal(false))

		Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(4 * time.Minute))
		Expect(config.BrokerStartTimeoutDuration()).To(Equal(10 * time.Minute))
		Expect(config.CfPushTimeoutDuration()).To(Equal(8 * time.Minute))
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

		Expect(config.GetPublicDockerAppImage()).To(Equal("cloudfoundry/diego-docker-app:latest"))
		Expect(config.GetUnallocatedIPForSecurityGroup()).To(Equal("10.0.244.255"))

		Expect(config.GetCredHubBrokerClientCredential()).To(Equal("credhub_admin_client"))
		Expect(config.GetCredHubLocation()).To(Equal("https://credhub.service.cf.internal:8844"))

		Expect(config.GetStacks()).To(ConsistOf("cflinuxfs4"))

		Expect(config.GetBinaryBuildpackName()).To(Equal("binary_buildpack"))
		Expect(config.GetGoBuildpackName()).To(Equal("go_buildpack"))
		Expect(config.GetHwcBuildpackName()).To(Equal("hwc_buildpack"))
		Expect(config.GetJavaBuildpackName()).To(Equal("java_buildpack"))
		Expect(config.GetNginxBuildpackName()).To(Equal("nginx_buildpack"))
		Expect(config.GetNodejsBuildpackName()).To(Equal("nodejs_buildpack"))
		Expect(config.GetRBuildpackName()).To(Equal("r_buildpack"))
		Expect(config.GetRubyBuildpackName()).To(Equal("ruby_buildpack"))
		Expect(config.GetStaticFileBuildpackName()).To(Equal("staticfile_buildpack"))
		Expect(config.GetPythonBuildpackName()).To(Equal("python_buildpack"))
	})

	Context("when all values are null", func() {
		It("returns an error", func() {
			nullConfigFilePath := writeConfigFile(&nullConfig{})
			_, err := cfg.NewCatsConfig(nullConfigFilePath)
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

			Expect(err.Error()).To(ContainSubstring("'isolation_segment_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'isolation_segment_domain' must not be null"))

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

			Expect(err.Error()).To(ContainSubstring("'credhub_mode' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'binary_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'go_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'java_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'nodejs_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'ruby_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'staticfile_buildpack_name' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'python_buildpack_name' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'include_apps' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_detect' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_docker' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_file_based_service_bindings' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_internet_dependent' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_private_docker_registry' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_route_services' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_routing' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_container_networking' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_sso' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_security_groups' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_services' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_user_provided_services' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_service_instance_sharing' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_ssh' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_tasks' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_http2_routing' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_tcp_routing' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_v3' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_zipkin' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_isolation_segments' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_routing_isolation_segments' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_tcp_isolation_segments' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_windows' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_service_discovery' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'include_app_syslog_tcp' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'public_docker_app_image' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'private_docker_registry_image' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'private_docker_registry_username' must not be null"))
			Expect(err.Error()).To(ContainSubstring("'private_docker_registry_password' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'name_prefix' must not be null"))

			Expect(err.Error()).To(ContainSubstring("'stacks' must not be null"))

			// These values are allowed to be null
			Expect(err.Error()).NotTo(ContainSubstring("unallocated_ip_for_security_group"))
			Expect(err.Error()).NotTo(ContainSubstring("use_windows_context_path"))
			Expect(err.Error()).NotTo(ContainSubstring("reporter_config"))
			Expect(err.Error()).NotTo(ContainSubstring("use_windows_test_task"))
			Expect(err.Error()).NotTo(ContainSubstring("include_volume_services"))
			Expect(err.Error()).NotTo(ContainSubstring("include_deployments"))
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

			testCfg.IncludeAppSyslogTcp = ptrToBool(false)
			testCfg.IncludeApps = ptrToBool(false)
			testCfg.IncludeContainerNetworking = ptrToBool(true)
			testCfg.IncludeDeployments = ptrToBool(true)
			testCfg.IncludeDetect = ptrToBool(false)
			testCfg.IncludeDocker = ptrToBool(true)
			testCfg.IncludeFileBasedServiceBindings = ptrToBool(true)
			testCfg.IncludeIPv6 = ptrToBool(true)
			testCfg.IncludeInternetDependent = ptrToBool(true)
			testCfg.IncludeIsolationSegments = ptrToBool(true)
			testCfg.IncludePrivateDockerRegistry = ptrToBool(true)
			testCfg.IncludeRouteServices = ptrToBool(true)
			testCfg.IncludeRouting = ptrToBool(false)
			testCfg.IncludeRoutingIsolationSegments = ptrToBool(true)
			testCfg.IncludeSSO = ptrToBool(true)
			testCfg.IncludeSecurityGroups = ptrToBool(true)
			testCfg.IncludeServiceDiscovery = ptrToBool(true)
			testCfg.IncludeServiceInstanceSharing = ptrToBool(true)
			testCfg.IncludeServices = ptrToBool(true)
			testCfg.IncludeUserProvidedServices = ptrToBool(true)
			testCfg.IncludeSsh = ptrToBool(true)
			testCfg.IncludeTCPIsolationSegments = ptrToBool(true)
			testCfg.IncludeHTTP2Routing = ptrToBool(true)
			testCfg.IncludeTCPRouting = ptrToBool(true)
			testCfg.IncludeTasks = ptrToBool(true)
			testCfg.IncludeV3 = ptrToBool(false)
			testCfg.IncludeVolumeServices = ptrToBool(true)
			testCfg.IncludeZipkin = ptrToBool(true)
			testCfg.IncludeWindows = ptrToBool(true)

			testCfg.BinaryBuildpackName = ptrToString("binary_buildpack_override")
			testCfg.GoBuildpackName = ptrToString("go_buildpack_override")
			testCfg.HwcBuildpackName = ptrToString("hwc_buildpack_override")
			testCfg.JavaBuildpackName = ptrToString("java_buildpack_override")
			testCfg.NginxBuildpackName = ptrToString("nginx_buildpack_override")
			testCfg.NodejsBuildpackName = ptrToString("nodejs_buildpack_override")
			testCfg.RBuildpackName = ptrToString("r_buildpack_override")
			testCfg.RubyBuildpackName = ptrToString("ruby_buildpack_override")
			testCfg.StaticFileBuildpackName = ptrToString("staticfile_buildpack_override")
			testCfg.PythonBuildpackName = ptrToString("python_buildpack_override")

			// These values are set so as not to trigger validation errors associated with the overrides provided above
			testCfg.PrivateDockerRegistryImage = ptrToString("avoid-validation-errors-by-setting-dummy-value")
			testCfg.PrivateDockerRegistryUsername = ptrToString("avoid-validation-errors-by-setting-dummy-value")
			testCfg.PrivateDockerRegistryPassword = ptrToString("avoid-validation-errors-by-setting-dummy-value")
			testCfg.IsolationSegmentName = ptrToString("avoid-validation-errors-by-setting-dummy-value")
			testCfg.IsolationSegmentDomain = ptrToString("avoid-validation-errors-by-setting-dummy-value")
			testCfg.VolumeServiceName = ptrToString("avoid-validation-errors-by-setting-dummy-value")
			testCfg.VolumeServicePlanName = ptrToString("avoid-validation-errors-by-setting-dummy-value")
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

			Expect(config.GetIncludeAppSyslogTcp()).To(BeFalse())
			Expect(config.GetIncludeApps()).To(BeFalse())
			Expect(config.GetIncludeContainerNetworking()).To(BeTrue())
			Expect(config.GetIncludeDeployments()).To(BeTrue())
			Expect(config.GetIncludeDetect()).To(BeFalse())
			Expect(config.GetIncludeDocker()).To(BeTrue())
			Expect(config.GetIncludeFileBasedServiceBindings()).To(BeTrue())
			Expect(config.GetIncludeIPv6()).To(BeTrue())
			Expect(config.GetIncludeInternetDependent()).To(BeTrue())
			Expect(config.GetIncludeIsolationSegments()).To(BeTrue())
			Expect(config.GetIncludePrivateDockerRegistry()).To(BeTrue())
			Expect(config.GetIncludeRouteServices()).To(BeTrue())
			Expect(config.GetIncludeRouting()).To(BeFalse())
			Expect(config.GetIncludeRoutingIsolationSegments()).To(BeTrue())
			Expect(config.GetIncludeSSO()).To(BeTrue())
			Expect(config.GetIncludeSecurityGroups()).To(BeTrue())
			Expect(config.GetIncludeServiceDiscovery()).To(BeTrue())
			Expect(config.GetIncludeServiceInstanceSharing()).To(BeTrue())
			Expect(config.GetIncludeServices()).To(BeTrue())
			Expect(config.GetIncludeUserProvidedServices()).To(BeTrue())
			Expect(config.GetIncludeSsh()).To(BeTrue())
			Expect(config.GetIncludeTCPIsolationSegments()).To(BeTrue())
			Expect(config.GetIncludeHTTP2Routing()).To(BeTrue())
			Expect(config.GetIncludeTCPRouting()).To(BeTrue())
			Expect(config.GetIncludeTasks()).To(BeTrue())
			Expect(config.GetIncludeV3()).To(BeFalse())
			Expect(config.GetIncludeVolumeServices()).To(BeTrue())
			Expect(config.GetIncludeZipkin()).To(BeTrue())
			Expect(config.GetIncludeWindows()).To(BeTrue())

			Expect(config.GetBinaryBuildpackName()).To(Equal("binary_buildpack_override"))
			Expect(config.GetGoBuildpackName()).To(Equal("go_buildpack_override"))
			Expect(config.GetHwcBuildpackName()).To(Equal("hwc_buildpack_override"))
			Expect(config.GetJavaBuildpackName()).To(Equal("java_buildpack_override"))
			Expect(config.GetNginxBuildpackName()).To(Equal("nginx_buildpack_override"))
			Expect(config.GetNodejsBuildpackName()).To(Equal("nodejs_buildpack_override"))
			Expect(config.GetRBuildpackName()).To(Equal("r_buildpack_override"))
			Expect(config.GetRubyBuildpackName()).To(Equal("ruby_buildpack_override"))
			Expect(config.GetStaticFileBuildpackName()).To(Equal("staticfile_buildpack_override"))
			Expect(config.GetPythonBuildpackName()).To(Equal("python_buildpack_override"))
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
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'private_docker_registry_image' must be provided if 'include_private_docker_registry' is true"))
			})
		})

		Context("when username is an empty string", func() {
			BeforeEach(func() {
				testCfg.PrivateDockerRegistryUsername = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'private_docker_registry_username' must be provided if 'include_private_docker_registry' is true"))
			})
		})

		Context("when password is an empty string", func() {
			BeforeEach(func() {
				testCfg.PrivateDockerRegistryPassword = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
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
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'public_docker_app_image' must be set to a valid image source"))
			})
		})
	})

	Context("when including catnip_docker_app_image", func() {
		Context("when image name is set", func() {
			var image = "some-image"
			BeforeEach(func() {
				testCfg.CatnipDockerAppImage = ptrToString(image)
			})

			It("has the value in the config", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.GetCatnipDockerAppImage()).To(Equal(image))
			})
		})

		Context("when image is an empty string", func() {
			BeforeEach(func() {
				testCfg.CatnipDockerAppImage = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'catnip_docker_app_image' must be set to a valid image source"))
			})
		})
	})

	Context("when including isolation segment tests", func() {
		BeforeEach(func() {
			testCfg.IncludeIsolationSegments = ptrToBool(true)
			testCfg.IsolationSegmentName = ptrToString("value")
		})

		Context("when name is an empty string", func() {
			BeforeEach(func() {
				testCfg.IsolationSegmentName = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'isolation_segment_name' must be provided if 'include_isolation_segments' is true"))
			})
		})
	})

	Context("when including windows tests", func() {
		BeforeEach(func() {
			testCfg.IncludeWindows = ptrToBool(true)
		})

		Context("when use_windows_context_path is set", func() {
			BeforeEach(func() {
				testCfg.UseWindowsContextPath = ptrToBool(true)
			})

			It("is loaded into the config", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(config.GetUseWindowsContextPath()).To(BeTrue())
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
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'isolation_segment_name' must be provided if 'include_routing_isolation_segments' is true"))
			})
		})

		Context("when isolation_segment_domain is an empty string", func() {
			BeforeEach(func() {
				testCfg.IsolationSegmentDomain = ptrToString("")
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* Invalid configuration: 'isolation_segment_domain' must be provided if 'include_routing_isolation_segments' is true"))
			})
		})
	})

	Context("when providing any set of stacks in the stacks property", func() {
		BeforeEach(func() {
			testCfg.Stacks = &[]string{"cflinuxfs4", "my-custom-stack"}
		})

		It("is loaded into the config", func() {
			config, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(config.GetStacks()).To(Equal([]string{"cflinuxfs4", "my-custom-stack"}))
		})
	})

	Context("when including a reporter config", func() {
		BeforeEach(func() {
			reporterConfig := &testReporterConfig{
				HoneyCombWriteKey: "some-write-key",
				HoneyCombDataset:  "some-dataset",
			}
			testCfg.ReporterConfig = reporterConfig
		})

		It("is loaded into the config", func() {
			config, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).ToNot(HaveOccurred())

			testReporterConfig := config.GetReporterConfig()
			Expect(testReporterConfig.HoneyCombWriteKey).To(Equal("some-write-key"))
			Expect(testReporterConfig.HoneyCombDataset).To(Equal("some-dataset"))
		})
		Context("when the reporter config includes custom tags", func() {
			BeforeEach(func() {
				customTags := map[string]interface{}{
					"some-tag": "some-tag-value",
				}
				testCfg.ReporterConfig.CustomTags = customTags
			})
			It("is loaded into the config", func() {
				config, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).ToNot(HaveOccurred())

				testReporterConfig := config.GetReporterConfig()
				Expect(testReporterConfig.CustomTags).To(Equal(map[string]interface{}{
					"some-tag": "some-tag-value",
				}))
			})
		})
	})

	Context("when including a timeout scale", func() {
		Context("when the timeout scale is zero", func() {
			BeforeEach(func() {
				testCfg.TimeoutScale = ptrToFloat(0.0)
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* 'timeout_scale' must be greater than zero"))
			})
		})

		Context("when the timeout scale is less than zero", func() {
			BeforeEach(func() {
				testCfg.TimeoutScale = ptrToFloat(-1.0)
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(MatchError("* 'timeout_scale' must be greater than zero"))
			})
		})
	})

	Describe("error aggregation", func() {
		BeforeEach(func() {
			testCfg.AdminPassword = nil
			testCfg.ApiEndpoint = ptrToString("invalid-url.asdf")
		})

		It("aggregates all errors", func() {
			_, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("* 'admin_password' must not be null"))
			Expect(err.Error()).To(ContainSubstring("* Invalid configuration for 'api' <invalid-url.asdf>"))
		})
	})

	Describe("GetApiEndpoint", func() {
		It(`returns the URL`, func() {
			cfg, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.GetApiEndpoint()).To(Equal("api." + BoshLiteDomain))
		})

		Context("when url is an IP address", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("10.244.0.34") // api.bosh-lite.env.wg-ard.ci.cloudfoundry.org
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
				Expect(err).To(MatchError("* Invalid configuration: 'api' must be a valid domain but was set to '_bogus%%%'"))
			})
		})

		Context("when the url contains https://", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ptrToString("https://api." + BoshLiteDomain)
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'api' must not contain a scheme/protocol but was set to 'https' in 'https://api." + BoshLiteDomain + "'"))
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
			Expect(c.GetAppsDomain()).To(Equal("cf-app." + BoshLiteDomain))
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
