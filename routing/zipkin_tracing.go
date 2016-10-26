package routing

import (
	"fmt"
	"regexp"
	"strconv"

	"code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	cf_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = ZipkinDescribe("Zipkin Tracing", func() {

	var (
		app1              string
		helloRoutingAsset = assets.NewAssets().SpringSleuthZip
		hostname          string
	)

	BeforeEach(func() {
		app1 = random_name.CATSRandomName("APP")
		helpers.PushApp(app1, helloRoutingAsset, Config.GetJavaBuildpackName(), Config.GetAppsDomain(), Config.CfPushTimeoutDuration(), "1G")

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

				appLogsSession := cf.Cf("logs", "--recent", app1)

				Eventually(appLogsSession.Out, "5s").Should(gbytes.Say("x_b3_traceid"))
				_, _, parentSpanId := grabIDs(string(appLogsSession.Out.Contents()), "")

				Expect(parentSpanId).To(Equal("-"))

				By("when request has zipkin trace headers")

				traceId := "fee1f7ba6aeec41c"

				header1 := fmt.Sprintf(`X-B3-TraceId: %s `, traceId)
				header2 := `X-B3-SpanId: 579b36fd31cd8714`
				Eventually(func() string {
					curlOutput = cf_helpers.CurlApp(Config, hostname, "/", "-H", header1, "-H", header2)
					return curlOutput
				}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("parents:"))

				appLogsSession = cf.Cf("logs", "--recent", hostname)

				Eventually(appLogsSession.Out).Should(gbytes.Say("x_b3_traceid:\"fee1f7ba6aeec41c"))
				_, appLogSpanId, _ := grabIDs(string(appLogsSession.Out.Contents()), traceId)

				Expect(curlOutput).To(ContainSubstring(traceId))
				Expect(curlOutput).To(ContainSubstring(fmt.Sprintf("parents: [%s]", appLogSpanId)))

			})
		})
	})
})

func grabIDs(logLines string, traceId string) (string, string, string) {
	defer GinkgoRecover()
	var re *regexp.Regexp

	if traceId == "" {
		re = regexp.MustCompile("x_b3_traceid:\"([0-9a-fA-F]*)\" x_b3_spanid:\"([0-9a-fA-F]*)\" x_b3_parentspanid:\"([0-9a-fA-F-]*)\"")
	} else {
		regex := fmt.Sprintf("x_b3_traceid:\"(%s)\" x_b3_spanid:\"([0-9a-fA-F]*)\" x_b3_parentspanid:\"([0-9a-fA-F-]*)\"", traceId)
		re = regexp.MustCompile(regex)
	}
	matches := re.FindStringSubmatch(logLines)

	Expect(matches).To(HaveLen(4))

	// traceid, spanid, parentspanid
	trimmedMatches, err := trimZeros(matches[1:])
	Expect(err).ToNot(HaveOccurred())
	return trimmedMatches[0], trimmedMatches[1], trimmedMatches[2]
}

func trimZeros(in []string) ([]string, error) {
	var out []string
	for _, s := range in {
		if s != "-" {
			x, err := strconv.ParseUint(s, 16, 64)
			if err != nil {
				return nil, err
			}
			out = append(out, fmt.Sprintf("%x", x))
		} else {
			out = append(out, "-")
		}
	}
	return out, nil
}
