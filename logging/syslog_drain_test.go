package logging

import (
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Logging", func() {
	var logWriterAppName string
	var listenerAppName string
	var logs *Session
	interrupt := make(chan string)
	serviceName := "service-" + generator.RandomName()

	Describe("Syslog drains", func() {
		BeforeEach(func() {
			listenerAppName = generator.PrefixedRandomName("CATS-APP-")
			logWriterAppName = generator.PrefixedRandomName("CATS-APP-")

			Eventually(cf.Cf("push", listenerAppName, "--no-start", "--health-check-type", "port", "-b", config.GoBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().SyslogDrainListener, "-d", config.AppsDomain, "-f", assets.NewAssets().SyslogDrainListener+"/manifest.yml"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to push app")
			Eventually(cf.Cf("push", logWriterAppName, "--no-start", "-b", config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().RubySimple, "-d", config.AppsDomain), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to push app")

			app_helpers.SetBackend(listenerAppName)
			app_helpers.SetBackend(logWriterAppName)

			Expect(cf.Cf("start", listenerAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
			logs = cf.Cf("logs", listenerAppName)
			Expect(cf.Cf("start", logWriterAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			syslogDrainURL := "syslog://" + getSyslogDrainAddress(listenerAppName)

			Eventually(cf.Cf("cups", serviceName, "-l", syslogDrainURL), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to create syslog drain service")
			Eventually(cf.Cf("bind-service", logWriterAppName, serviceName), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to bind service")
		})

		AfterEach(func() {
			interrupt <- "done"

			app_helpers.AppReport(logWriterAppName, DEFAULT_TIMEOUT)
			app_helpers.AppReport(listenerAppName, DEFAULT_TIMEOUT)

			Eventually(cf.Cf("delete", logWriterAppName, "-f", "-r"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to delete app")
			Eventually(cf.Cf("delete", listenerAppName, "-f", "-r"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to delete app")
			if serviceName != "" {
				Eventually(cf.Cf("delete-service", serviceName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0), "Failed to delete service")
			}

			Eventually(cf.Cf("delete-orphaned-routes", "-f"), CF_PUSH_TIMEOUT).Should(Exit(0), "Failed to delete orphaned routes")
		})

		It("forwards app messages to registered syslog drains", func() {
			randomMessage := "random-message-" + generator.RandomName()
			go writeLogsUntilInterrupted(interrupt, randomMessage, logWriterAppName)

			Eventually(logs, (DEFAULT_TIMEOUT + time.Minute)).Should(Say(randomMessage))
		})
	})
})

func getSyslogDrainAddress(appName string) string {
	config := helpers.LoadConfig()

	if config.Backend == "dea" {
		cfApp := cf.Cf("files", appName, "/app/address")
		Eventually(cfApp, DEFAULT_TIMEOUT).Should(Exit(0))
		lines := strings.Split(string(cfApp.Out.Contents()), "\n")
		return lines[len(lines)-2]
	}

	cfApp := cf.Cf("ssh", appName, "-c", "echo $CF_INSTANCE_ADDR")
	Eventually(cfApp, DEFAULT_TIMEOUT).Should(Exit(0))
	return strings.TrimSpace(string(cfApp.Out.Contents()))
}

func writeLogsUntilInterrupted(interrupt chan string, randomMessage string, logWriterAppName string) {
	for {
		select {
		case <-interrupt:
			return
		default:
			helpers.CurlAppWithTimeout(logWriterAppName, "/log/"+randomMessage, DEFAULT_TIMEOUT)
			time.Sleep(3 * time.Second)
		}
	}
}
