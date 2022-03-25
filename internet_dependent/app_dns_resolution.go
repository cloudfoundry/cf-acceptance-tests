package internet_dependent_test

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-test-helpers/helpers"
)

type CatnipCurlResponse struct {
	Stdout     string
	Stderr     string
	ReturnCode int `json:"return_code"`
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
		app_helpers.AppReport(clientAppName)
		Expect(cf.Cf("delete", clientAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	})

	It("allows app containers to resolve public DNS", func() {
		clientAppName = random_name.CATSRandomName("APP")

		Expect(cf.Cf(app_helpers.CatnipWithArgs(clientAppName, "-m", DEFAULT_MEMORY_LIMIT)...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("Connecting from running container to an external destination")
		catnipCurlResponse = testAppConnectivity(clientAppName, "www.google.com", 80)
		Expect(catnipCurlResponse.ReturnCode).To(Equal(0), "Expected external traffic to be allowed from app containers to external addresses.")
	})
})
