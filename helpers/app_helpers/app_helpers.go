package app_helpers

import (
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/lager"
)

func GetAppGuid(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Eventually(cfApp, Config.DefaultTimeoutDuration()).Should(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func SetBackend(appName string) {
	if Config.GetBackend() == "diego" {
		EnableDiego(appName)
	} else if Config.GetBackend() == "dea" {
		DisableDiego(appName)
	}
}

func EnableDiego(appName string) {
	guid := GetAppGuid(appName)
	Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": true}`), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func DisableDiego(appName string) {
	guid := GetAppGuid(appName)
	Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": false}`), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func DisableDiegoAndCheckResponse(appName, expectedSubstring string) {
	guid := GetAppGuid(appName)
	Eventually(func() string {
		response := cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego":false}`)
		Expect(response.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		return string(response.Out.Contents())
	}, Config.DefaultTimeoutDuration(), "1s").Should(ContainSubstring(expectedSubstring))
}

func AppReport(appName string, timeout time.Duration) {
	if appName == "" {
		return
	}
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(Exit())
	Eventually(cf.Cf("logs", appName, "--recent"), timeout).Should(Exit())
}

func DeleteLRP(appGuid string, logger lager.Logger) error {
	bbsClient, err := XXX.NewBBSClient()
	if err != nil {
		return err
	}
	desiredLRPs, err := bbsClient.DesiredLRPs(logger, nil)
	if err != nil {
		return err
	}
	processGuids := []string{} // for error messages only
	for _, desiredLRP := range desiredLRPs {
		processGuid := desiredLRP.ProcessGuid
		if strings.Index(processGuid, appGuid) == 0 {
			bbsClient.RemoveDesiredLRP(logger, processGuid)
			return nil
		}
		processGuids = append(processGuids, processGuid)
	}
	return fmt.Error("DeleteLRP: Couldn't find a desiredLRP starting with appGuid:%s (processGuids:%s)", appGuid, processGuids)
}
