package apps

import (
	"fmt"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
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

var _ = Describe("Uploading and Downloading droplets", func() {
	var appName string
	var otherAppName string

	AfterEach(func() {
		app_helpers.AppReport(appName)
		app_helpers.AppReport(otherAppName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", otherAppName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("Users can manage droplet bits for an app", func() {
		By("Pushing an app with 'hello world' in the response")
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName, "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().HelloWorld).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}).Should(ContainSubstring("Hello, world!"))

		By("Pushing other app with 'healthy' in the response")
		otherAppName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", otherAppName, "-p", assets.NewAssets().RubySimple).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, otherAppName)
		}).Should(ContainSubstring("Healthy"))

		By("Downloading the droplet for the Hello World app")
		guid := appGuid(appName)
		tmpdir := makeTempDir()
		fmt.Println("Tmpdir:" + tmpdir)
		defer os.RemoveAll(tmpdir)

		appDroplet := app_helpers.NewAppDroplet(guid, Config)
		appDropletPath := path.Join(tmpdir, appName)
		appDropletPathToCompressedFile, err := appDroplet.DownloadTo(appDropletPath)

		Expect(err).ToNot(HaveOccurred())
		unpackTarball(appDropletPathToCompressedFile)

		By("Creating an empty droplet for the 'healthy' app")
		otherAppGuid := appGuid(otherAppName)
		emptyDroplet := app_helpers.CreateEmptyDroplet(otherAppGuid)

		By("Uploading the 'hello world' tgz to the empty droplet")
		emptyDroplet.UploadFrom(appDropletPathToCompressedFile)

		By("Setting the other app droplet to the hello world droplet")
		emptyDroplet.SetAsCurrentDroplet()

		By("Restarting the healthy app and confirming it says 'hello world'")
		Expect(cf.Cf("restart", otherAppName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, otherAppName)
		}).Should(ContainSubstring("Hello, world!"))
	})
})
