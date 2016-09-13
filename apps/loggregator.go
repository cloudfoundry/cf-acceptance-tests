package apps

import (
	"fmt"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/matchers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/noaa"
	"github.com/cloudfoundry/noaa/events"

	"crypto/tls"
	"strings"

	"encoding/json"
	"os"
	"path/filepath"
)

var _ = AppsDescribe("loggregator", func() {
	var appName string
	const hundredthOfOneSecond = 10000 // this app uses millionth of seconds

	BeforeEach(func() {
		appName = CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-b", config.RubyBuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().LoggregatorLoadGenerator,
			"-d", Config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Context("cf logs", func() {
		var logs *Session

		BeforeEach(func() {
			logs = cf.Cf("logs", appName)
		})

		AfterEach(func() {
			// logs might be nil if the BeforeEach panics
			if logs != nil {
				logs.Interrupt().Wait(DEFAULT_TIMEOUT)
			}
		})

		It("exercises basic loggregator behavior", func() {
			Eventually(logs, (DEFAULT_TIMEOUT + time.Minute)).Should(Say("Connected, tailing logs for app"))

			Eventually(func() string {
				return helpers.CurlApp(appName, fmt.Sprintf("/log/sleep/%d", hundredthOfOneSecond))
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Muahaha"))

			Eventually(logs, (DEFAULT_TIMEOUT + time.Minute)).Should(Say("Muahaha"))
		})
	})

	Context("cf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			Eventually(func() string {
				return helpers.CurlApp(appName, fmt.Sprintf("/log/sleep/%d", hundredthOfOneSecond))
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Muahaha"))

			Eventually(func() *Session {
				appLogsSession := cf.Cf("logs", "--recent", appName)
				Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				return appLogsSession
			}, DEFAULT_TIMEOUT).Should(Say("Muahaha"))
		})
	})

	Context("firehose data", func() {
		It("shows logs and metrics", func() {
			noaaConnection := noaa.NewConsumer(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: config.SkipSSLValidation}, nil)
			msgChan := make(chan *events.Envelope, 100000)
			errorChan := make(chan error)
			stopchan := make(chan struct{})

			go noaaConnection.Firehose(CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken(), msgChan, errorChan, stopchan)
			defer close(stopchan)

			Eventually(func() string {
				return helpers.CurlApp(appName, fmt.Sprintf("/log/sleep/%d", hundredthOfOneSecond))
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Muahaha"))

			Eventually(msgChan, 10*time.Second).Should(Receive(EnvelopeContainingMessageLike("Muahaha")), "To enable the logging & metrics firehose feature, please ask your CF administrator to add the 'doppler.firehose' scope to your CF admin user.")
		})

		It("shows container metrics", func() {
			appGuid := strings.TrimSpace(string(cf.Cf("app", appName, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()))

			noaaConnection := noaa.NewConsumer(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: config.SkipSSLValidation}, nil)
			msgChan := make(chan *events.Envelope, 100000)
			errorChan := make(chan error)
			stopchan := make(chan struct{})
			go noaaConnection.Firehose(CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken(), msgChan, errorChan, stopchan)
			defer close(stopchan)

			Eventually(func() bool {
				for {
					select {
					case msg := <-msgChan:
						if cm := msg.GetContainerMetric(); cm != nil {
							if cm.GetApplicationId() == appGuid && cm.GetInstanceIndex() == 0 {
								return true
							}
						}
					case e := <-errorChan:
						Expect(e).ToNot(HaveOccurred())
					default:
						return false
					}
				}
			}, 2*DEFAULT_TIMEOUT).Should(BeTrue())
		})
	})
})

type cfHomeConfig struct {
	AccessToken         string
	LoggregatorEndpoint string
}

func getCfHomeConfig() *cfHomeConfig {
	myCfHomeConfig := &cfHomeConfig{}

	workflowhelpers.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		path := filepath.Join(os.Getenv("CF_HOME"), ".cf", "config.json")

		configFile, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		decoder := json.NewDecoder(configFile)
		err = decoder.Decode(myCfHomeConfig)
		if err != nil {
			panic(err)
		}
	})

	return myCfHomeConfig
}

func getAdminUserAccessToken() string {
	return getCfHomeConfig().AccessToken
}

func getDopplerEndpoint() string {
	return strings.Replace(getCfHomeConfig().LoggregatorEndpoint, "loggregator", "doppler", -1)
}
