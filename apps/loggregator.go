package apps

import (
	"fmt"

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
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/matchers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/noaa"
	"github.com/cloudfoundry/noaa/events"

	"crypto/tls"
	"strings"

	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var _ = AppsDescribe("loggregator", func() {
	var appName string
	const hundredthOfOneSecond = 10000 // this app uses millionth of seconds

	BeforeEach(func() {
		appName = CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().LoggregatorLoadGenerator,
			"-i", "2",
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("cf logs", func() {
		var logs *Session

		BeforeEach(func() {
			logs = logshelper.TailFollow(Config.GetUseLogCache(), appName)
		})

		AfterEach(func() {
			// logs might be nil if the BeforeEach panics
			if logs != nil {
				logs.Interrupt()
			}
		})

		It("exercises basic loggregator behavior", func() {
			if !Config.GetUseLogCache() {
				// log cache cli will not emit header unless being run in terminal
				Eventually(logs).Should(Say("(Connected, tailing|Retrieving) logs for app"))
			}

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", hundredthOfOneSecond))
			}).Should(ContainSubstring("Muahaha"))

			Eventually(logs).Should(Say("Muahaha"))
		})
	})

	Context("cf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", hundredthOfOneSecond))
			}).Should(ContainSubstring("Muahaha"))

			Eventually(logshelper.Tail(Config.GetUseLogCache(), appName)).Should(Say("Muahaha"))
		})
	})

	Context("firehose data", func() {
		It("shows logs and metrics", func() {
			noaaConnection := noaa.NewConsumer(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: Config.GetSkipSSLValidation()}, nil)
			msgChan := make(chan *events.Envelope, 100000)
			errorChan := make(chan error)
			stopchan := make(chan struct{})

			go noaaConnection.Firehose(CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken(), msgChan, errorChan, stopchan)
			defer close(stopchan)

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, fmt.Sprintf("/log/sleep/%d", hundredthOfOneSecond))
			}).Should(ContainSubstring("Muahaha"))

			Eventually(msgChan, 2*Config.DefaultTimeoutDuration(), time.Millisecond).Should(Receive(EnvelopeContainingMessageLike("Muahaha")), "To enable the logging & metrics firehose feature, please ask your CF administrator to add the 'doppler.firehose' scope to your CF admin user.")
		})

		It("shows container metrics", func() {
			appGuid := strings.TrimSpace(string(cf.Cf("app", appName, "--guid").Wait().Out.Contents()))

			noaaConnection := noaa.NewConsumer(getDopplerEndpoint(), &tls.Config{InsecureSkipVerify: Config.GetSkipSSLValidation()}, nil)
			msgChan := make(chan *events.Envelope, 100000)
			errorChan := make(chan error)
			stopchan := make(chan struct{})
			go noaaConnection.Firehose(CATSRandomName("SUBSCRIPTION-ID"), getAdminUserAccessToken(), msgChan, errorChan, stopchan)
			defer close(stopchan)

			Eventually(msgChan, 2*Config.DefaultTimeoutDuration(), time.Millisecond).Should(Receive(NonZeroContainerMetricsFor(MetricsApp{AppGuid: appGuid, InstanceId: 0})))
			Eventually(msgChan, 2*Config.DefaultTimeoutDuration(), time.Millisecond).Should(Receive(NonZeroContainerMetricsFor(MetricsApp{AppGuid: appGuid, InstanceId: 1})))
		})
	})
})

type cfHomeConfig struct {
	AccessToken     string
	DopplerEndPoint string
}

func getCfHomeConfig() *cfHomeConfig {
	myCfHomeConfig := &cfHomeConfig{}

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
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
	return getCfHomeConfig().DopplerEndPoint
}
