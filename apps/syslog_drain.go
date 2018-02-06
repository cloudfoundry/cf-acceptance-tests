package apps

import (
	"regexp"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Logging", func() {
	var logWriterAppName1 string
	var logWriterAppName2 string
	var listenerAppName string
	var logs *Session
	var interrupt chan struct{}
	var serviceName string

	Describe("Syslog drains", func() {
		BeforeEach(func() {
			interrupt = make(chan struct{}, 1)
			serviceName = random_name.CATSRandomName("SVIN")
			listenerAppName = random_name.CATSRandomName("APP")
			logWriterAppName1 = random_name.CATSRandomName("APP")
			logWriterAppName2 = random_name.CATSRandomName("APP")

			Eventually(cf.Cf(
				"push",
				listenerAppName,
				"--no-start",
				"--health-check-type", "port",
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().SyslogDrainListener,
				"-d", Config.GetAppsDomain(),
				"-f", assets.NewAssets().SyslogDrainListener+"/manifest.yml",
			), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to push app")

			Eventually(cf.Cf(
				"push",
				logWriterAppName1,
				"--no-start",
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().RubySimple,
				"-d", Config.GetAppsDomain(),
			), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to push app")

			Eventually(cf.Cf(
				"push",
				logWriterAppName2,
				"--no-start",
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().RubySimple,
				"-d", Config.GetAppsDomain(),
			), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to push app")

			app_helpers.SetBackend(listenerAppName)
			app_helpers.SetBackend(logWriterAppName1)
			app_helpers.SetBackend(logWriterAppName2)

			Expect(cf.Cf("start", listenerAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("start", logWriterAppName1).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("start", logWriterAppName2).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			logs.Kill()
			close(interrupt)

			app_helpers.AppReport(logWriterAppName1, Config.DefaultTimeoutDuration())
			app_helpers.AppReport(logWriterAppName2, Config.DefaultTimeoutDuration())
			app_helpers.AppReport(listenerAppName, Config.DefaultTimeoutDuration())

			Eventually(cf.Cf("delete", logWriterAppName1, "-f", "-r"), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to delete app")
			Eventually(cf.Cf("delete", logWriterAppName2, "-f", "-r"), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to delete app")
			Eventually(cf.Cf("delete", listenerAppName, "-f", "-r"), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to delete app")
			if serviceName != "" {
				Eventually(cf.Cf("delete-service", serviceName, "-f"), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to delete service")
			}

			Eventually(cf.Cf("delete-orphaned-routes", "-f"), Config.CfPushTimeoutDuration()).Should(Exit(0), "Failed to delete orphaned routes")
		})

		It("forwards app messages to registered syslog drains", func() {
			syslogDrainURL := "syslog://" + getSyslogDrainAddress(listenerAppName)

			Eventually(cf.Cf("cups", serviceName, "-l", syslogDrainURL), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to create syslog drain service")
			Eventually(cf.Cf("bind-service", logWriterAppName1, serviceName), Config.DefaultTimeoutDuration()).Should(Exit(0), "Failed to bind service")
			// We don't need to restage, because syslog service bindings don't change the app's environment variables

			randomMessage1 := random_name.CATSRandomName("RANDOM-MESSAGE-A")
			randomMessage2 := random_name.CATSRandomName("RANDOM-MESSAGE-B")

			logs = logshelper.TailFollow(Config.GetUseLogCache(), listenerAppName)

			// Have apps emit logs.
			go writeLogsUntilInterrupted(interrupt, randomMessage1, logWriterAppName1)
			go writeLogsUntilInterrupted(interrupt, randomMessage2, logWriterAppName2)

			Eventually(logs, Config.DefaultTimeoutDuration()+2*time.Minute).Should(Say(randomMessage1))
			Consistently(logs, 10).ShouldNot(Say(randomMessage2))
		})
	})
})

func getSyslogDrainAddress(appName string) string {
	var address []byte

	Eventually(func() []byte {
		re, err := regexp.Compile("ADDRESS: \\|(.*)\\|")
		Expect(err).NotTo(HaveOccurred())

		logs := logshelper.Tail(Config.GetUseLogCache(), appName).
			Wait(Config.DefaultTimeoutDuration())
		matched := re.FindSubmatch(logs.Out.Contents())
		if len(matched) < 2 {
			return nil
		}
		address = matched[1]
		return address
	}, Config.DefaultTimeoutDuration()).Should(Not(BeNil()))

	return string(address)
}

func writeLogsUntilInterrupted(interrupt chan struct{}, randomMessage string, logWriterAppName string) {
	defer GinkgoRecover()
	for {
		select {
		case <-interrupt:
			return
		default:
			helpers.CurlAppWithTimeout(Config, logWriterAppName, "/log/"+randomMessage, Config.DefaultTimeoutDuration())
			time.Sleep(3 * time.Second)
		}
	}
}
