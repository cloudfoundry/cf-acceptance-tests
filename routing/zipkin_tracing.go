package routing

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	cf_helpers "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = ZipkinDescribe("Zipkin Tracing", func() {
	var (
		appName   string
		assetPath = assets.NewAssets().SpringSleuthZip
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetJavaBuildpackName(),
			"-m", "1024M",
			"-p", assetPath,
		).Wait(CF_JAVA_TIMEOUT)).To(gexec.Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(gexec.Exit(0))
	})

	Context("when Zipkin headers are not included in a request to an app", func() {
		It("GoRouter adds some Zipkin headers", func() {
			Eventually(func() string {
				return cf_helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("parents:"))

			Eventually(func() *gbytes.Buffer {
				return logs.Recent(appName).Wait().Out
			}).Should(And(gbytes.Say(`x_b3_spanid:"([0-9a-fA-F]*)"`), gbytes.Say(`x_b3_parentspanid:"-"`)))
		})
	})

	Context("when Zipkin headers are included in a request to a Zipkin-enabled app", func() {
		var (
			traceID = "fee1f7ba6aeec41c"
			spanID  = "579b36fd31cd8714"
		)

		It("GoRouter passes the provided Zipkin headers through", func() {
			var (
				traceHeader = fmt.Sprintf("X-B3-TraceId: %s", traceID)
				spanHeader  = fmt.Sprintf("X-B3-SpanId: %s", spanID)
			)
			Eventually(func() string {
				return cf_helpers.CurlApp(Config, appName, "/", "-H", traceHeader, "-H", spanHeader)
			}).Should(MatchRegexp("current span: [Trace: %s.*parents: [%s]", traceID, spanID))

			var (
				traceRegex = `x_b3_traceid:"%s"`
				spanRegex  = `x_b3_spanid:"%s"`
			)
			Eventually(func() *gbytes.Buffer {
				return logs.Recent(appName).Wait().Out
			}).Should(And(gbytes.Say(traceRegex, traceID), gbytes.Say(spanRegex, spanID)))
		})
	})
})
