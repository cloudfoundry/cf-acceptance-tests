package app_helpers

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func GetAppGuid(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Eventually(cfApp).Should(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func AppReport(appName string) {
	if appName == "" {
		return
	}

	printStartAppReport(appName)

	Eventually(cf.Cf("app", appName, "--guid")).Should(Exit())
	Eventually(logs.Tail(Config.GetUseLogCache(), appName)).Should(Exit())

	printEndAppReport(appName)
}

func printStartAppReport(appName string) {
	printAppReportBanner(fmt.Sprintf("***** APP REPORT: %s *****", appName))
}

func printEndAppReport(appName string) {
	printAppReportBanner(fmt.Sprintf("*** END APP REPORT: %s ***", appName))
}

func printAppReportBanner(announcement string) {
	startColor, endColor := getColor()
	sequence := strings.Repeat("*", len(announcement))
	fmt.Fprintf(ginkgo.GinkgoWriter,
		"\n\n%s%s\n%s\n%s%s\n",
		startColor,
		sequence,
		announcement,
		sequence,
		endColor)
}

func getColor() (string, string) {
	startColor := ""
	endColor := ""
	if !config.DefaultReporterConfig.NoColor {
		startColor = "\x1b[35m"
		endColor = "\x1b[0m"
	}

	return startColor, endColor
}
