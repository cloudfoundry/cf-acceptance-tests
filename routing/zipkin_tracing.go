package routing

import (
	"fmt"
	"regexp"

	"code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	cf_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = ZipkinDescribe("Zipkin Tracing", func() {
	var (
		app1              string
		helloRoutingAsset = assets.NewAssets().SpringSleuthZip
		hostname          string
	)

	BeforeEach(func() {
		app1 = random_name.CATSRandomName("APP")
		helpers.PushApp(app1, helloRoutingAsset, Config.GetJavaBuildpackName(), Config.GetAppsDomain(), CF_JAVA_TIMEOUT, "1024M")

		hostname = app1
	})

	AfterEach(func() {
		helpers.AppReport(app1, Config.DefaultTimeoutDuration())
		helpers.DeleteApp(app1, Config.DefaultTimeoutDuration())
	})

	Context("when zipkin tracing is enabled", func() {
		Context("when zipkin headers are not in the request", func() {
			It("the sleuth error response has no error", func() {
				var curlOutput string
				// when req does not have headers
				Eventually(func() string {
					curlOutput = cf_helpers.CurlAppRoot(Config, hostname)
					return curlOutput
				}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("parents:"))

				appLogsSession := logs.Tail(Config.GetUseLogCache(), app1)

				Eventually(appLogsSession, Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))

				Eventually(appLogsSession.Out).Should(gbytes.Say("x_b3_traceid"))
				parentSpanID := getID(`x_b3_parentspanid:"([0-9a-fA-F-]*)"`, string(appLogsSession.Out.Contents()))

				Expect(parentSpanID).To(Equal("-"))

				By("when request has zipkin trace headers")

				traceID := "fee1f7ba6aeec41c"

				header1 := fmt.Sprintf(`X-B3-TraceId: %s `, traceID)
				header2 := `X-B3-SpanId: 579b36fd31cd8714`
				Eventually(func() string {
					curlOutput = cf_helpers.CurlApp(Config, hostname, "/", "-H", header1, "-H", header2)
					return curlOutput
				}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("parents:"))

				appLogsSession = logs.Tail(Config.GetUseLogCache(), hostname)

				Eventually(appLogsSession, Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))

				Expect(appLogsSession.Out).To(gbytes.Say(`x_b3_traceid:"fee1f7ba6aeec41c"`))

				spanIDRegex := fmt.Sprintf("x_b3_traceid:\"%s\" x_b3_spanid:\"([0-9a-fA-F]*)\"", traceID)

				appLogSpanID := getID(spanIDRegex, string(appLogsSession.Out.Contents()))

				Expect(curlOutput).To(ContainSubstring(traceID))
				Expect(curlOutput).To(ContainSubstring(fmt.Sprintf("parents: [%s]", appLogSpanID)))
			})
		})
	})
})

func getID(logRegex, logLines string) string {
	matches := regexp.MustCompile(logRegex).FindStringSubmatch(logLines)
	Expect(matches).To(HaveLen(2))

	return matches[1]
}
