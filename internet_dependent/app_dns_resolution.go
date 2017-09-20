package internet_dependent_test

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

type CatnipCurlResponse struct {
	Stdout     string
	Stderr     string
	ReturnCode int `json:"return_code"`
}

func pushApp(appName, buildpack string) {
	Expect(cf.Cf("push",
		appName,
		"--no-start",
		"-b", buildpack,
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", assets.NewAssets().Catnip,
		"-c", "./catnip",
		"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	app_helpers.SetBackend(appName)
}

func testAppConnectivity(clientAppName string, privateHost string, privatePort int) CatnipCurlResponse {
	var catnipCurlResponse CatnipCurlResponse
	curlResponse := helpers.CurlApp(Config, clientAppName, fmt.Sprintf("/curl/%s/%d", privateHost, privatePort))
	json.Unmarshal([]byte(curlResponse), &catnipCurlResponse)
	return catnipCurlResponse
}

var _ = InternetDependentDescribe("App container DNS behavior", func() {
	var clientAppName string
	var catnipCurlResponse CatnipCurlResponse

	BeforeEach(func() {
		if !Config.GetIncludeInternetDependent() {
			Skip(skip_messages.SkipInternetDependentMessage)
		}

	})

	AfterEach(func() {
		app_helpers.AppReport(clientAppName, Config.DefaultTimeoutDuration())
		Expect(cf.Cf("delete", clientAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	})

	It("allows app containers to resolve public DNS", func() {
		clientAppName = random_name.CATSRandomName("APP")
		pushApp(clientAppName, Config.GetBinaryBuildpackName())
		Expect(cf.Cf("start", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("Connecting from running container to an external destination")
		catnipCurlResponse = testAppConnectivity(clientAppName, "www.google.com", 80)
		Expect(catnipCurlResponse.ReturnCode).To(Equal(0), "Expected external traffic to be allowed from app containers to external addresses.")
	})
})
