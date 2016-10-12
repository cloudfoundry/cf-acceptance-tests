package config_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	cfg "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/validationerrors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type requiredConfig struct {
	// required
	ApiEndpoint       string `json:"api"`
	AdminUser         string `json:"admin_user"`
	AdminPassword     string `json:"admin_password"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	AppsDomain        string `json:"apps_domain"`
	UseHttp           bool   `json:"use_http"`
}

type testConfig struct {
	// required
	ApiEndpoint       string `json:"api"`
	AdminUser         string `json:"admin_user"`
	AdminPassword     string `json:"admin_password"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	AppsDomain        string `json:"apps_domain"`
	UseHttp           bool   `json:"use_http"`

	// timeouts
	DefaultTimeout               int `json:"default_timeout"`
	CfPushTimeout                int `json:"cf_push_timeout"`
	LongCurlTimeout              int `json:"long_curl_timeout"`
	BrokerStartTimeout           int `json:"broker_start_timeout"`
	AsyncServiceOperationTimeout int `json:"async_service_operation_timeout"`
	DetectTimeout                int `json:"detect_timeout"`
	SleepTimeout                 int `json:"sleep_timeout"`

	// optional
	Backend string `json:"backend"`
}

var tmpFile *os.File
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

func withConfig(initialConfig testConfig, setupConfig func(testConfig) testConfig, runTest func()) {
	previousConfig := os.Getenv("CONFIG")
	updatedConfig := setupConfig(initialConfig)

	newConfigFilePath := writeConfigFile(updatedConfig)
	os.Setenv("CONFIG", newConfigFilePath)

	runTest()
	os.Setenv("CONFIG", previousConfig)
}

var _ = Describe("Config", func() {
	BeforeEach(func() {
		testCfg = testConfig{
			ApiEndpoint:       "api.bosh-lite.com",
			AdminUser:         "admin",
			AdminPassword:     "admin",
			SkipSSLValidation: true,
			AppsDomain:        "cf-app.bosh-lite.com",
			UseHttp:           true,
		}
		requiredCfg = requiredConfig{
			ApiEndpoint:       "api.bosh-lite.com",
			AdminUser:         "admin",
			AdminPassword:     "admin",
			SkipSSLValidation: true,
			AppsDomain:        "cf-app.bosh-lite.com",
			UseHttp:           true,
		}

		tmpFile, err = ioutil.TempFile("", "cf-test-helpers-config")
		Expect(err).NotTo(HaveOccurred())

		encoder := json.NewEncoder(tmpFile)
		err = encoder.Encode(requiredCfg)
		Expect(err).NotTo(HaveOccurred())

		err = tmpFile.Close()
		Expect(err).NotTo(HaveOccurred())

		originalConfig = os.Getenv("CONFIG")
		os.Setenv("CONFIG", tmpFile.Name())
	})

	AfterEach(func() {
		err := os.Remove(tmpFile.Name())
		Expect(err).NotTo(HaveOccurred())

		os.Setenv("CONFIG", originalConfig)
	})

	It("should have the right defaults", func() {
		config, err := cfg.NewCatsConfig()
		Expect(err).ToNot(HaveOccurred())
		Expect(config.GetIncludeApps()).To(BeTrue())
		Expect(config.DefaultTimeoutDuration()).To(Equal(30 * time.Second))
		Expect(config.CfPushTimeoutDuration()).To(Equal(2 * time.Minute))
		Expect(config.LongCurlTimeoutDuration()).To(Equal(2 * time.Minute))
		Expect(config.BrokerStartTimeoutDuration()).To(Equal(5 * time.Minute))
		Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(2 * time.Minute))

		// undocumented
		Expect(config.DetectTimeoutDuration()).To(Equal(5 * time.Minute))
		Expect(config.SleepTimeoutDuration()).To(Equal(30 * time.Second))
	})

	It("should have duration timeouts based on the configured values", func() {
		withConfig(testCfg, func(myConfig testConfig) testConfig {
			myConfig.DefaultTimeout = 12
			myConfig.CfPushTimeout = 34
			myConfig.LongCurlTimeout = 56
			myConfig.BrokerStartTimeout = 78
			myConfig.AsyncServiceOperationTimeout = 90
			myConfig.DetectTimeout = 100
			myConfig.SleepTimeout = 101
			return myConfig
		},
			func() {
				config, err := cfg.NewCatsConfig()
				Expect(err).NotTo(HaveOccurred())

				Expect(config.DefaultTimeoutDuration()).To(Equal(12 * time.Second))
				Expect(config.CfPushTimeoutDuration()).To(Equal(34 * time.Minute))
				Expect(config.LongCurlTimeoutDuration()).To(Equal(56 * time.Minute))
				Expect(config.BrokerStartTimeoutDuration()).To(Equal(78 * time.Minute))
				Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(90 * time.Minute))
				Expect(config.DetectTimeoutDuration()).To(Equal(100 * time.Minute))
				Expect(config.SleepTimeoutDuration()).To(Equal(101 * time.Second))
			},
		)
	})

	Context(`validations`, func() {
		It(`validates that the backend is "dea", "diego", or ""`, func() {
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.Backend = "lkjlkjlkjlkj"
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("* Invalid configuration: 'backend' must be 'diego', 'dea', or empty but was set to 'lkjlkjlkjlkj'"))
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.Backend = "dea"
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).NotTo(HaveOccurred())
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.Backend = "diego"
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).NotTo(HaveOccurred())
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.Backend = ""
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).NotTo(HaveOccurred())
				},
			)
		})

		It(`validates that ApiEndpoint is a valid URL`, func() {
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myAlwaysResolvingDomain := "api.bosh-lite.com"
				myConfig.ApiEndpoint = myAlwaysResolvingDomain
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).NotTo(HaveOccurred())
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myAlwaysResolvingIP := "10.244.0.34" // api.bosh-lite.com
				myConfig.ApiEndpoint = myAlwaysResolvingIP
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).NotTo(HaveOccurred())
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.ApiEndpoint = ""
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank"))
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.ApiEndpoint = "_bogus%%%"
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("* Invalid configuration: 'api' must be a valid URL but was set to '_bogus%%%'"))
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myNeverResolvingURI := "E437FE20-5F25-479E-8B79-A008A13E58F6.E437FE20-5F25-479E-8B79-A008A13E58F6.E437FE20-5F25-479E-8B79-A008A13E58F6"
				myConfig.ApiEndpoint = myNeverResolvingURI
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("no such host"))
				},
			)
		})

		It(`validates that AppsDomain is a valid domain`, func() {
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.AppsDomain = "bosh-lite.com"
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).ToNot(HaveOccurred())
				},
			)
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myAlwaysResolvingIP := "10.244.0.34" // api.bosh-lite.com
				myConfig.AppsDomain = myAlwaysResolvingIP
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("no such host"))
				},
			)
		})

		It(`validates that AdminUser is present`, func() {
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.AdminUser = ""
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("'admin_user' must be provided"))
				},
			)
		})

		It(`validates that AdminPassword is present`, func() {
			withConfig(testCfg, func(myConfig testConfig) testConfig {
				myConfig.AdminPassword = ""
				return myConfig
			},
				func() {
					_, err := cfg.NewCatsConfig()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("'admin_password' must be provided"))
				},
			)
		})
	})
})
