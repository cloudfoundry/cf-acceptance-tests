package ssh

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe(deaUnsupportedTag+"SSH", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		Eventually(cf.Cf(
			"push", appName,
			"-p", assets.NewAssets().Dora,
			"--no-start",
			"-b", "ruby_buildpack",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-d", config.AppsDomain,
			"-i", "1"),
			DEFAULT_TIMEOUT,
		).Should(Exit(0))

		app_helpers.SetBackend(appName)

		enableSSH(appName)

		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	Describe("ssh", func() {
		Context("with multiple instances", func() {
			BeforeEach(func() {
				Eventually(cf.Cf("scale", appName, "-i", "2"), CF_PUSH_TIMEOUT).Should(Exit(0))
				Eventually(func() string {
					return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
				}, DEFAULT_TIMEOUT).Should(Equal("1"))
			})

			It("can ssh to the second instance", func() {
				envCmd := cf.Cf("ssh", "-v", "-i", "1", appName, "-c", "/usr/bin/env && /usr/bin/env >&2")
				Expect(envCmd.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				output := string(envCmd.Out.Contents())
				stdErr := string(envCmd.Err.Contents())

				Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
				Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

				Expect(string(stdErr)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
				Expect(string(stdErr)).To(MatchRegexp("INSTANCE_INDEX=1"))

				Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Say("Successful remote access"))
				Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-authorized"))
			})
		})

		It("can execute a remote command in the container", func() {
			envCmd := cf.Cf("ssh", "-v", appName, "-c", "/usr/bin/env && /usr/bin/env >&2")
			Expect(envCmd.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			output := string(envCmd.Out.Contents())
			stdErr := string(envCmd.Err.Contents())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Expect(string(stdErr)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(stdErr)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Say("Successful remote access"))
			Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-authorized"))
		})

		It("runs an interactive session when no command is provided", func() {
			envCmd := exec.Command("cf", "ssh", "-v", appName)

			stdin, err := envCmd.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			stdout, err := envCmd.StdoutPipe()
			Expect(err).NotTo(HaveOccurred())

			err = envCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			_, err = stdin.Write([]byte("/usr/bin/env\n"))
			Expect(err).NotTo(HaveOccurred())

			err = stdin.Close()
			Expect(err).NotTo(HaveOccurred())

			output, err := ioutil.ReadAll(stdout)
			Expect(err).NotTo(HaveOccurred())

			err = envCmd.Wait()
			Expect(err).NotTo(HaveOccurred())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Say("Successful remote access"))
			Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-authorized"))
		})

		It("allows local port forwarding", func() {
			listenCmd := exec.Command("cf", "ssh", "-v", "-L", "127.0.0.1:61007:localhost:8080", appName)

			stdin, err := listenCmd.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			err = listenCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				curl := runner.Curl("http://127.0.0.1:61007/").Wait(DEFAULT_TIMEOUT)
				return string(curl.Out.Contents())
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora"))

			err = stdin.Close()
			Expect(err).NotTo(HaveOccurred())

			err = listenCmd.Wait()
			Expect(err).NotTo(HaveOccurred())
		})

		It("records successful ssh attempts", func() {
			password := sshAccessCode()

			clientConfig := &ssh.ClientConfig{
				User: fmt.Sprintf("cf:%s/%d", guidForAppName(appName), 0),
				Auth: []ssh.AuthMethod{ssh.Password(password)},
			}

			client, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).NotTo(HaveOccurred())

			session, err := client.NewSession()
			Expect(err).NotTo(HaveOccurred())

			output, err := session.CombinedOutput("/usr/bin/env")
			Expect(err).NotTo(HaveOccurred())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Say("Successful remote access"))
			Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-authorized"))
		})

		It("records failed ssh attempts", func() {
			Eventually(cf.Cf("disable-ssh", appName), DEFAULT_TIMEOUT).Should(Exit(0))

			password := sshAccessCode()
			clientConfig := &ssh.ClientConfig{
				User: fmt.Sprintf("cf:%s/%d", guidForAppName(appName), 0),
				Auth: []ssh.AuthMethod{ssh.Password(password)},
			}

			_, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).To(HaveOccurred())

			Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-unauthorized"))
		})
	})

})

func enableSSH(appName string) {
	Eventually(cf.Cf("enable-ssh", appName), DEFAULT_TIMEOUT).Should(Exit(0))
}

func sshAccessCode() string {
	getCode := cf.Cf("ssh-code")
	Eventually(getCode, DEFAULT_TIMEOUT).Should(Exit(0))
	return strings.TrimSpace(string(getCode.Buffer().Contents()))
}

func sayCommandRun(cmd *exec.Cmd) {
	const timeFormat = "2006-01-02 15:04:05.00 (MST)"

	startColor := ""
	endColor := ""
	if !ginkgoconfig.DefaultReporterConfig.NoColor {
		startColor = "\x1b[32m"
		endColor = "\x1b[0m"
	}
	fmt.Fprintf(GinkgoWriter, "\n%s[%s]> %s %s\n", startColor, time.Now().UTC().Format(timeFormat), strings.Join(cmd.Args, " "), endColor)
}

func sshProxyAddress() string {
	infoCommand := cf.Cf("curl", "/v2/info")
	Expect(infoCommand.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	type infoResponse struct {
		AppSSHEndpoint string `json:"app_ssh_endpoint"`
	}

	var response infoResponse
	err := json.Unmarshal(infoCommand.Buffer().Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	return response.AppSSHEndpoint
}
