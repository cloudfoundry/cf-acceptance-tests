package apps

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Downloading droplets", func() {
	var helloWorldAppName string
	var out bytes.Buffer

	BeforeEach(func() {
		helloWorldAppName = generator.PrefixedRandomName("CATS-APP-")

		Expect(cf.Cf("push", helloWorldAppName, "-m", "128M", "-p", assets.NewAssets().HelloWorld, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", helloWorldAppName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Downloads the droplet for the app", func() {
		guid := cf.Cf("app", helloWorldAppName, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()
		appGuid := strings.TrimSpace(string(guid))

		tmpdir, err := ioutil.TempDir(os.TempDir(), "droplet-download")
		Expect(err).ToNot(HaveOccurred())

		app_droplet_path := path.Join(tmpdir, helloWorldAppName)

		cf.Cf("curl", fmt.Sprintf("/v2/apps/%s/droplet/download", appGuid), "--output", app_droplet_path).Wait(DEFAULT_TIMEOUT)

		cmd := exec.Command("tar", "-ztf", app_droplet_path)
		cmd.Stdout = &out
		err = cmd.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(out.String()).To(ContainSubstring("./app/config.ru"))
		Expect(out.String()).To(ContainSubstring("./tmp"))
		Expect(out.String()).To(ContainSubstring("./logs"))
	})
})
