package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

var _ = AppsDescribe("Uploading and Downloading droplets", func() {
	var helloWorldAppName string
	var out bytes.Buffer

	BeforeEach(func() {
		helloWorldAppName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push", helloWorldAppName, "--no-start", "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().HelloWorld, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(helloWorldAppName)
		Expect(cf.Cf("start", helloWorldAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(helloWorldAppName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", helloWorldAppName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("Users can manage droplet bits for an app", func() {
		By("Downloading the droplet for the app")

		guid := cf.Cf("app", helloWorldAppName, "--guid").Wait(Config.DefaultTimeoutDuration()).Out.Contents()
		appGuid := strings.TrimSpace(string(guid))

		tmpdir, err := ioutil.TempDir(os.TempDir(), "droplet-download")
		Expect(err).ToNot(HaveOccurred())

		app_droplet_path := path.Join(tmpdir, helloWorldAppName)

		cf.Cf("curl", fmt.Sprintf("/v2/apps/%s/droplet/download", appGuid), "--output", app_droplet_path).Wait(Config.DefaultTimeoutDuration())

		cmd := exec.Command("tar", "-ztf", app_droplet_path)
		cmd.Stdout = &out
		err = cmd.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(out.String()).To(ContainSubstring("./app/config.ru"))
		Expect(out.String()).To(ContainSubstring("./tmp"))
		Expect(out.String()).To(ContainSubstring("./logs"))

		By("Pushing a different version of the app")

		Expect(cf.Cf("push", helloWorldAppName, "-p", assets.NewAssets().Dora).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

		By("Uploading the originally downloaded droplet of the app")

		token := v3_helpers.GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v2/apps/%s/droplet/upload", Config.Protocol(), Config.GetApiEndpoint(), appGuid)
		bits := fmt.Sprintf(`droplet=@%s`, app_droplet_path)
		curl := helpers.Curl(Config, "-v", uploadUrl, "-X", "PUT", "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait(Config.DefaultTimeoutDuration())
		Expect(curl).To(Exit(0))

		var job struct {
			Metadata struct {
				Url string `json:"url"`
			} `json:"metadata"`
		}
		bytes := curl.Out.Contents()
		json.Unmarshal(bytes, &job)
		pollingUrl := job.Metadata.Url

		Eventually(func() *Session {
			return cf.Cf("curl", pollingUrl).Wait(Config.DefaultTimeoutDuration())
		}, Config.DefaultTimeoutDuration()).Should(Say("finished"))

		By("Running the original droplet for the app")

		cf.Cf("restart", helloWorldAppName).Wait(Config.DefaultTimeoutDuration())

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, world!"))
	})
})
