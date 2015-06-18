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
		helloWorldAppName = generator.RandomName()

		Expect(cf.Cf("push", helloWorldAppName, "-p", assets.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", helloWorldAppName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Downloads the droplet for the app", func() {
		guid := cf.Cf("app", helloWorldAppName, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()
		appGuid := strings.TrimSpace(string(guid))

		cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps/%s/droplet/download", appGuid)).Wait(DEFAULT_TIMEOUT).Out.Contents()

		tmpdir, err := ioutil.TempDir(os.TempDir(), "droplet-download")
		Expect(err).ToNot(HaveOccurred())

		app_droplet_path := path.Join(tmpdir, helloWorldAppName)

		f, err := os.Create(app_droplet_path)
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()

		bytes_written, err := f.Write(cfResponse)
		Expect(err).ToNot(HaveOccurred())
		Expect(bytes_written).To(BeNumerically(">", 0))

		cmd := exec.Command("tar", "-ztf", app_droplet_path)
		cmd.Stdout = &out
		err = cmd.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(out.String()).To(ContainSubstring("./app/config.ru"))
		Expect(out.String()).To(ContainSubstring("./logs/staging_task.log"))

	})
})
