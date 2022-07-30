package v3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/download"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = V3Describe("package features", func() {
	var (
		appName            string
		appGuid            string
		packageGuid        string
		spaceGuid          string
		destinationAppGuid string
		token              string
		uploadUrl          string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, "{}")
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl = fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		DeleteApp(appGuid)
	})

	Context("with a valid package", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		AfterEach(func() {
			if destinationAppGuid != "" {
				DeleteApp(destinationAppGuid)
			}
		})

		It("can copy package bits to another app and download the package", func() {
			destinationAppName := random_name.CATSRandomName("APP")
			destinationAppGuid = CreateApp(destinationAppName, spaceGuid, "{}")

			// COPY
			copyRequestBody := fmt.Sprintf("{\"relationships\":{\"app\":{\"data\":{\"guid\":\"%s\"}}}}", destinationAppGuid)
			copyUrl := fmt.Sprintf("v3/packages/?source_guid=%s", packageGuid)

			session := cf.Cf("curl", copyUrl, "-X", "POST", "-d", copyRequestBody)
			bytes := session.Wait().Out.Contents()
			var pac struct {
				Guid string `json:"guid"`
			}
			json.Unmarshal(bytes, &pac)
			copiedPackageGuid := pac.Guid

			WaitForPackageToBeReady(copiedPackageGuid)

			tmpdir, err := ioutil.TempDir(os.TempDir(), "package-download")
			Expect(err).ToNot(HaveOccurred())
			app_package_path := path.Join(tmpdir, destinationAppName)

			// DOWNLOAD
			downloadURL := fmt.Sprintf("/v3/packages/%s/download", copiedPackageGuid)
			err = download.WithRedirect(downloadURL, app_package_path, Config)
			Expect(err).ToNot(HaveOccurred())

			session = helpers.Run("unzip", "-l", app_package_path)
			Expect(session.Wait()).To(Exit(0))
			Expect(session.Out).To(Say("dora.rb"))
		})
	})

	Context("when the package contains files in unwriteable directories", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().JavaUnwriteableZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		It("can still stage the package", func() {
			buildGuid := StageBuildpackPackage(packageGuid, Config.GetJavaBuildpackName())
			buildPath := fmt.Sprintf("/v3/builds/%s", buildGuid)

			Eventually(func() *Session {
				return cf.Cf("curl", buildPath).Wait()
			}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
		})
	})
})
