package apps

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Copy app bits", func() {
	var golangAppName string
	var helloWorldAppName string
	type AppResource struct {
		Metadata struct {
			GUID string `json:"guid"`
		} `json:"metadata"`
	}

	type AppsResponse struct {
		Resources []AppResource `json:"resources"`
	}

	BeforeEach(func() {
		golangAppName = generator.RandomName()
		helloWorldAppName = generator.RandomName()

		Expect(cf.Cf("push", golangAppName, "-p", assets.NewAssets().Golang, "--no-start").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("push", helloWorldAppName, "-p", assets.NewAssets().HelloWorld, "--no-start").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", golangAppName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("delete", helloWorldAppName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Copies over the package from the source app to the destination app", func() {
		var golangAppsResponse AppsResponse
		var helloWorldAppsResponse AppsResponse

		golangResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", golangAppName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
		json.Unmarshal(golangResponse, &golangAppsResponse)
		helloWorldResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", helloWorldAppName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
		json.Unmarshal(helloWorldResponse, &helloWorldAppsResponse)

		requestURL := fmt.Sprintf("/v2/apps/%s/copy_bits", golangAppsResponse.Resources[0].Metadata.GUID)
		requestBody := fmt.Sprintf(`{"source_app_guid": "%s"}`, helloWorldAppsResponse.Resources[0].Metadata.GUID)

		Expect(cf.Cf("curl", requestURL, "-X", "POST", "-d", requestBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("restart", helloWorldAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(helloWorldAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, world!"))
	})
})
