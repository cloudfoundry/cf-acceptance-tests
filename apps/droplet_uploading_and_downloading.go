package apps

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

func appGuid(appName string) string {
	guid := cf.Cf("app", appName, "--guid").Wait().Out.Contents()
	return strings.TrimSpace(string(guid))
}

func makeTempDir() string {
	// This defers from the typical way temp directories are created (using the OS default), because the GNUWin32 tar.exe does not allow file paths to be prefixed with a drive letter.
	tmpdir, err := ioutil.TempDir(".", "droplet-download")
	Expect(err).ToNot(HaveOccurred())
	return tmpdir
}

func unpackTarball(tarballPath string) {
	// The gzip and tar commands have been tested and works in both Linux and Windows environments. In Windows, it was tested using GNUWin32 executables. The reason why this is split into two steps instead of running 'tar -ztf file_name' is because the GNUWin32 tar.exe does not support '-z'.
	cmd := exec.Command("gzip", "-dk", tarballPath)
	session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(Exit(0))

	cmd = exec.Command("tar", "-tf", strings.Trim(tarballPath, ".gz"))
	session, err = Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(Exit(0))

	Expect(session.Out.Contents()).To(ContainSubstring("./app/config.ru"))
	Expect(session.Out.Contents()).To(ContainSubstring("./tmp"))
	Expect(session.Out.Contents()).To(ContainSubstring("./logs"))
}

var _ = AppsDescribe("Uploading and Downloading droplets", func() {
	var helloWorldAppName string

	BeforeEach(func() {
		helloWorldAppName = random_name.CATSRandomName("APP")

		Expect(cf.Push(helloWorldAppName, "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().HelloWorld, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(helloWorldAppName)

		Expect(cf.Cf("delete", helloWorldAppName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("Users can manage droplet bits for an app", func() {
		guid := appGuid(helloWorldAppName)
		tmpdir := makeTempDir()
		defer os.RemoveAll(tmpdir)

		By("Downloading the droplet for the app")
		appDroplet := app_helpers.NewAppDroplet(guid, Config)
		appDropletPath := path.Join(tmpdir, helloWorldAppName)
		appDropletPathToCompressedFile, err := appDroplet.DownloadTo(appDropletPath)
		Expect(err).ToNot(HaveOccurred())
		unpackTarball(appDropletPathToCompressedFile)

		By("Pushing a different version of the app")
		Expect(cf.Push(helloWorldAppName, "-p", assets.NewAssets().RubySimple).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}).Should(ContainSubstring("Healthy"))

		By("Uploading the originally downloaded droplet of the app")
		appDroplet.UploadFrom(appDropletPathToCompressedFile)

		By("Running the original droplet for the app")
		cf.Cf("restart", helloWorldAppName).Wait()

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, helloWorldAppName)
		}).Should(ContainSubstring("Hello, world!"))
	})
})
