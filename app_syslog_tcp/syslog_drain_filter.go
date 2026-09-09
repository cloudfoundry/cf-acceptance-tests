package apps

import (
	"fmt"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/tcp_routing"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

// Verifies the per-drain log source type filter from the syslog-agent. HTTP traffic
// against a ruby_simple app produces both APP logs (the appMarker echoed to STDOUT)
// and RTR logs (gorouter access logs).
const rtrSourceTypeMarker = `source_type="RTR"`

var _ = AppSyslogTcpDescribe("Syslog Drain source type filter over TCP", func() {
	var (
		logWriterAppName string
		externalPort     string
		domainName       string
		listenerAppName  string
		logs             *Session
		interrupt        chan struct{}
		serviceName      string
	)

	Describe("Syslog drain source type filter", Ordered, func() {
		BeforeAll(func() {
			domainName = Config.GetTCPDomain()
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				routerGroupOutput := string(cf.Cf("router-groups").Wait().Out.Contents())
				Expect(routerGroupOutput).To(
					MatchRegexp(fmt.Sprintf("%s\\s+tcp", tcp_routing.DefaultRouterGroupName)),
					fmt.Sprintf("Router group %s of type tcp doesn't exist", tcp_routing.DefaultRouterGroupName),
				)

				Expect(cf.Cf("create-shared-domain",
					domainName,
					"--router-group", tcp_routing.DefaultRouterGroupName,
				).Wait()).To(Exit())
			})
			listenerAppName = random_name.CATSRandomName("APP-SYSLOG-LISTENER")
			logWriterAppName = random_name.CATSRandomName("APP-LOG-WRITER")

			Eventually(cf.Cf(
				"push",
				listenerAppName,
				"--health-check-type", "port",
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().SyslogDrainListener,
				"-f", assets.NewAssets().SyslogDrainListener+"/manifest.yml",
			), Config.CfPushTimeoutDuration()).Should(Exit(0), "Failed to push listener app")

			externalPort = MapTCPRoute(listenerAppName, domainName)

			Eventually(cf.Cf(
				"push",
				logWriterAppName,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().RubySimple,
			), Config.CfPushTimeoutDuration()).Should(Exit(0), "Failed to push log writer app")
		})

		BeforeEach(func() {
			interrupt = make(chan struct{}, 1)
			serviceName = random_name.CATSRandomName("SVIN")
		})

		AfterEach(func() {
			if logs != nil {
				logs.Kill()
				logs = nil
			}
			if interrupt != nil {
				close(interrupt)
			}

			Eventually(cf.Cf("delete-service", serviceName, "-f")).Should(Exit(0), "Failed to delete service")
		})

		AfterAll(func() {
			app_helpers.AppReport(logWriterAppName)
			app_helpers.AppReport(listenerAppName)

			Eventually(cf.Cf("delete", logWriterAppName, "-f", "-r")).Should(Exit(0), "Failed to delete log writer app")
			Eventually(cf.Cf("delete", listenerAppName, "-f", "-r")).Should(Exit(0), "Failed to delete listener app")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", TestSetup.GetOrganizationName()).Wait()).To(Exit(0))
				Eventually(cf.Cf("delete-shared-domain", domainName, "-f")).Should(Exit(0), "Failed to delete TCP shared domain")
			})
			Eventually(cf.Cf("delete-orphaned-routes", "-f"), Config.CfPushTimeoutDuration()).Should(Exit(0), "Failed to delete orphaned routes")
		})

		assertDrainBehavior := func(drainURL, scenario string, expectRTR bool) {
			Eventually(cf.Cf("cups", serviceName, "-l", drainURL)).Should(Exit(0), "Failed to create syslog drain service")
			Eventually(cf.Cf("bind-service", logWriterAppName, serviceName)).Should(Exit(0), "Failed to bind service")

			appMarker := random_name.CATSRandomName("APP-MARKER")

			logs = logshelper.Follow(listenerAppName)

			go driveAppUntilInterrupted(interrupt, logWriterAppName, appMarker)

			Eventually(logs, Config.DefaultTimeoutDuration()+2*time.Minute).Should(Say(appMarker), "APP log line was not forwarded by the "+scenario+" drain")
			if expectRTR {
				Eventually(logs, Config.DefaultTimeoutDuration()+2*time.Minute).Should(Say(rtrSourceTypeMarker), "RTR log line was not forwarded by the "+scenario+" drain")
			} else {
				Consistently(logs, 30).ShouldNot(Say(rtrSourceTypeMarker), "RTR log line leaked through the "+scenario+" drain")
			}
		}

		It("include-log-types=APP forwards APP logs and drops RTR logs", func() {
			assertDrainBehavior(fmt.Sprintf("syslog://%s:%s/?include-log-types=APP", domainName, externalPort), "include-APP", false)
		})

		It("exclude-log-types=RTR forwards APP logs and drops RTR logs", func() {
			assertDrainBehavior(fmt.Sprintf("syslog://%s:%s/?exclude-log-types=RTR", domainName, externalPort), "exclude-RTR", false)
		})

		It("drain-data=all with include-log-types=APP forwards APP logs and drops RTR logs", func() {
			assertDrainBehavior(fmt.Sprintf("syslog://%s:%s/?drain-data=all&include-log-types=APP", domainName, externalPort), "drain-data=all include-APP", false)
		})

		It("include-log-types=STG,APP with two types forwards APP logs and drops RTR logs", func() {
			assertDrainBehavior(fmt.Sprintf("syslog://%s:%s/?include-log-types=STG,APP", domainName, externalPort), "include-STG,APP", false)
		})

		It("with no source type filter forwards both APP and RTR logs", func() {
			assertDrainBehavior(fmt.Sprintf("syslog://%s:%s/", domainName, externalPort), "unfiltered", true)
		})
	})
})

// driveAppUntilInterrupted curls the app so it emits both an APP log line
// (GET /log/<appMarker>) and RTR router access logs, until interrupted.
func driveAppUntilInterrupted(interrupt chan struct{}, appName, appMarker string) {
	defer GinkgoRecover()
	for {
		select {
		case <-interrupt:
			return
		default:
			helpers.CurlAppWithTimeout(Config, appName, "/log/"+appMarker, Config.DefaultTimeoutDuration())
			time.Sleep(3 * time.Second)
		}
	}
}
