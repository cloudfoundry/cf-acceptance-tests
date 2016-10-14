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
	})

	JustBeforeEach(func() {
		tmpFilePath = writeConfigFile(&testCfg)
	})

	AfterEach(func() {
		err := os.Remove(tmpFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should have the right defaults", func() {
		requiredCfg := requiredConfig{
			ApiEndpoint:       testCfg.ApiEndpoint,
			AdminUser:         testCfg.AdminUser,
			AdminPassword:     testCfg.AdminPassword,
			SkipSSLValidation: testCfg.SkipSSLValidation,
			AppsDomain:        testCfg.AppsDomain,
			UseHttp:           testCfg.UseHttp,
		}
		requiredCfgFilePath := writeConfigFile(requiredCfg)
		config, err := cfg.NewCatsConfig(requiredCfgFilePath)
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

	Context("when values with default are overriden", func() {
		BeforeEach(func() {
			testCfg.DefaultTimeout = 12
			testCfg.CfPushTimeout = 34
			testCfg.LongCurlTimeout = 56
			testCfg.BrokerStartTimeout = 78
			testCfg.AsyncServiceOperationTimeout = 90
			testCfg.DetectTimeout = 100
			testCfg.SleepTimeout = 101
		})

		It("respects the overriden values", func() {
			config, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(config.DefaultTimeoutDuration()).To(Equal(12 * time.Second))
			Expect(config.CfPushTimeoutDuration()).To(Equal(34 * time.Minute))
			Expect(config.LongCurlTimeoutDuration()).To(Equal(56 * time.Minute))
			Expect(config.BrokerStartTimeoutDuration()).To(Equal(78 * time.Minute))
			Expect(config.AsyncServiceOperationTimeoutDuration()).To(Equal(90 * time.Minute))
			Expect(config.DetectTimeoutDuration()).To(Equal(100 * time.Minute))
			Expect(config.SleepTimeoutDuration()).To(Equal(101 * time.Second))
		})
	})

	Describe(`GetBackend`, func() {
		Context("when the backend is set to `dea`", func() {
			BeforeEach(func() {
				testCfg.Backend = "dea"
			})

			It("returns `dea`", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetBackend()).To(Equal("dea"))
			})
		})

		Context("when the backend is set to `diego`", func() {
			BeforeEach(func() {
				testCfg.Backend = "diego"
			})

			It("returns `diego`", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetBackend()).To(Equal("diego"))
			})
		})

		Context("when the backend is empty", func() {
			BeforeEach(func() {
				testCfg.Backend = ""
			})

			It("returns an empty string", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetBackend()).To(Equal(""))
			})
		})

		Context("when the backend is set to any other value", func() {
			BeforeEach(func() {
				testCfg.Backend = "asdfasdf"
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
				testCfg.ApiEndpoint = "10.244.0.34" // api.bosh-lite.com
			})

			It("returns the IP address", func() {
				cfg, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.GetApiEndpoint()).To(Equal("10.244.0.34"))
			})
		})

		Context("when the domain does not resolve", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = "some-url-that-does-not-resolve.com"
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such host"))
			})
		})

		Context("when the url is empty", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = ""
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("* Invalid configuration: 'api' must be a valid Cloud Controller endpoint but was blank"))
			})
		})

		Context("when the url is invalid", func() {
			BeforeEach(func() {
				testCfg.ApiEndpoint = "_bogus%%%"
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'api' must be a valid URL but was set to '_bogus%%%'"))
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
				testCfg.AppsDomain = "_bogus%%%"
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("* Invalid configuration: 'apps_domain' must be a valid URL but was set to '_bogus%%%'"))
			})
		})

		Context("when the AppsDomain is an IP address (which is invalid for AppsDomain)", func() {
			BeforeEach(func() {
				testCfg.AppsDomain = "10.244.0.34"
			})

			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such host"))
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
				testCfg.AdminUser = ""
			})
			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_user' must be provided"))
			})
		})
	})

	Describe("GetAdminPassword", func() {
		It("returns the admin password", func() {
			c, err := cfg.NewCatsConfig(tmpFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.GetAdminPassword()).To(Equal("admin"))
		})

		Context("when the admin user is blank", func() {
			BeforeEach(func() {
				testCfg.AdminPassword = ""
			})
			It("returns an error", func() {
				_, err := cfg.NewCatsConfig(tmpFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'admin_password' must be provided"))
			})
		})
	})
})
