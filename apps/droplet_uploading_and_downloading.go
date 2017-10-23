package apps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
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

func appGuid(appName string) string {
	guid := cf.Cf("app", appName, "--guid").Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return strings.TrimSpace(string(guid))
}

func makeTempDir() string {
	// This defers from the typical way temp directories are created (using the OS default), because the GNUWin32 tar.exe does not allow file paths to be prefixed with a drive letter.
	tmpdir, err := ioutil.TempDir(".", "droplet-download")
	Expect(err).ToNot(HaveOccurred())
	return tmpdir
}

func curlAndFollowRedirectWithoutHeaders(downloadURL, appDropletPathToCompressedFile string) {
	oauthToken := v3_helpers.GetAuthToken()
	downloadCurl := helpers.Curl(
		Config,
		"-v", fmt.Sprintf("%s%s", Config.GetApiEndpoint(), downloadURL),
		"-H", fmt.Sprintf("Authorization: %s", oauthToken),
		"-f",
	).Wait(Config.DefaultTimeoutDuration())
	Expect(downloadCurl).To(Exit(0))

	curlOutput := string(downloadCurl.Err.Contents())
	locationHeaderRegex := regexp.MustCompile("Location: (.*)\r\n")
	redirectURI := locationHeaderRegex.FindStringSubmatch(curlOutput)[1]

	downloadCurl = helpers.Curl(
		Config,
		"-v", redirectURI,
		"--output", appDropletPathToCompressedFile,
		"-f",
	).Wait(Config.DefaultTimeoutDuration())
	Expect(downloadCurl).To(Exit(0))
}

func downloadDroplet(appGuid, downloadDirectory string) string {
	appDropletPathToCompressedFile := fmt.Sprintf("%s.tar.gz", downloadDirectory)
	downloadUrl := fmt.Sprintf("/v2/apps/%s/droplet/download", appGuid)

	curlAndFollowRedirectWithoutHeaders(downloadUrl, appDropletPathToCompressedFile)
	return appDropletPathToCompressedFile
}

func uploadDroplet(appGuid, dropletPath string) {
	token := v3_helpers.GetAuthToken()
	uploadUrl := fmt.Sprintf("%s%s/v2/apps/%s/droplet/upload", Config.Protocol(), Config.GetApiEndpoint(), appGuid)
	bits := fmt.Sprintf(`droplet=@%s`, dropletPath)
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
}

func unpackTarball(tarballPath string) {
	// The gzip and tar commands have been tested and works in both Linux and Windows environments. In Windows, it was tested using GNUWin32 executables. The reason why this is split into two steps instead of running 'tar -ztf file_name' is because the GNUWin32 tar.exe does not support '-z'.
	cmd := exec.Command("gzip", "-dk", tarballPath)
	session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, Config.DefaultTimeoutDuration()).Should(Exit(0))

	cmd = exec.Command("tar", "-tf", strings.Trim(tarballPath, ".gz"))
	session, err = Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, Config.DefaultTimeoutDuration()).Should(Exit(0))

	Expect(session.Out.Contents()).To(ContainSubstring("./app/config.ru"))
	Expect(session.Out.Contents()).To(ContainSubstring("./tmp"))
	Expect(session.Out.Contents()).To(ContainSubstring("./logs"))
}

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
		guid := appGuid(helloWorldAppName)
		tmpdir := makeTempDir()
		defer os.RemoveAll(tmpdir)

		By("Downloading the droplet for the app")
		appDropletPath := path.Join(tmpdir, helloWorldAppName)
		appDropletPathToCompressedFile := downloadDroplet(guid, appDropletPath)
		unpackTarball(appDropletPathToCompressedFile)

		By("Pushing a different version of the app")
		Expect(cf.Cf("push", helloWorldAppName, "-p", assets.NewAssets().RubySimple).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Healthy"))

		By("Uploading the originally downloaded droplet of the app")
		uploadDroplet(guid, appDropletPathToCompressedFile)

		By("Running the original droplet for the app")
		cf.Cf("restart", helloWorldAppName).Wait(Config.DefaultTimeoutDuration())

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, world!"))
	})
})
