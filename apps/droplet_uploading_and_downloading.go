package apps

import (
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

		// This defers from the typical way temp directories are created (using the OS default), because the GNUWin32 tar.exe does not allow file paths to be prefixed with a drive letter.
		tmpdir, err := ioutil.TempDir(".", "droplet-download")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpdir)

		app_droplet_path := path.Join(tmpdir, helloWorldAppName)
		app_droplet_path_to_tar_file := fmt.Sprintf("%s.tar", app_droplet_path)
		app_droplet_path_to_compressed_file := fmt.Sprintf("%s.tar.gz", app_droplet_path)

		cf.Cf("curl", fmt.Sprintf("/v2/apps/%s/droplet/download", appGuid), "--output", app_droplet_path_to_compressed_file).Wait(Config.DefaultTimeoutDuration())

		var session *Session

		// The gzip and tar commands have been tested and works in both Linux and Windows environments. In Windows, it was tested using GNUWin32 executables. The reason why this is split into two steps instead of running 'tar -ztf file_name' is because the GNUWin32 tar.exe does not support '-z'.
		cmd := exec.Command("gzip", "-dk", app_droplet_path_to_compressed_file)
		session, err = Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session, Config.DefaultTimeoutDuration()).Should(Exit(0))

		cmd = exec.Command("tar", "-tf", app_droplet_path_to_tar_file)
		session, err = Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session, Config.DefaultTimeoutDuration()).Should(Exit(0))

		Expect(session.Out.Contents()).To(ContainSubstring("./app/config.ru"))
		Expect(session.Out.Contents()).To(ContainSubstring("./tmp"))
		Expect(session.Out.Contents()).To(ContainSubstring("./logs"))

		By("Pushing a different version of the app")

		Expect(cf.Cf("push", helloWorldAppName, "-p", assets.NewAssets().Dora).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

		By("Uploading the originally downloaded droplet of the app")

		token := v3_helpers.GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v2/apps/%s/droplet/upload", Config.Protocol(), Config.GetApiEndpoint(), appGuid)
		bits := fmt.Sprintf(`droplet=@%s`, app_droplet_path_to_compressed_file)
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
