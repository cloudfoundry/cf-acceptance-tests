package app_helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
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
	if appName == "" || !ginkgo.CurrentGinkgoTestDescription().Failed {
		return
	}

	printStartAppReport(appName)

	Eventually(cf.Cf("app", appName, "--guid"), time.Second*60).Should(Exit())
	Eventually(logs.Recent(appName), time.Second*60).Should(Exit())

	printEndAppReport(appName)
}

func ReportedIDs(instances int, appName string) map[string]bool {
	timer := time.NewTimer(time.Second * 120)
	defer timer.Stop()
	run := true
	go func() {
		<-timer.C
		run = false
	}()

	seenIDs := map[string]bool{}
	for len(seenIDs) != instances && run == true {
		seenIDs[helpers.CurlApp(Config, appName, "/id")] = true
		time.Sleep(time.Second)
	}

	return seenIDs
}

func DifferentIDsFrom(idsBefore map[string]bool, appName string) []string {
	differentIDs := []string{}

	for id := range ReportedIDs(len(idsBefore), appName) {
		if !idsBefore[id] {
			differentIDs = append(differentIDs, id)
		}
	}

	return differentIDs
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
