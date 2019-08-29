package ssh

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"golang.org/x/crypto/ssh"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = SshDescribe("SSH", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Eventually(cf.Cf(
			"push", appName,
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Catnip,
			"-c", "./catnip",
		),
			Config.CfPushTimeoutDuration(),
		).Should(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Describe("ssh", func() {
		Context("with multiple instances", func() {
			BeforeEach(func() {
				Eventually(cf.Cf("scale", appName, "-i", "2")).Should(Exit(0))
			})

			It("can ssh to the second instance", func() {
				// sometimes ssh'ing to the second instance fails because the instance isn't running
				// so we try a few times
				Eventually(func() *Session {
					return cf.Cf("ssh", "-v", "-i", "1", appName, "-c", "/usr/bin/env && /usr/bin/env >&2").Wait()
				}).Should(Exit(0))

				// once we know that ssh can succeed we grab the output for checking
				envCmd := cf.Cf("ssh", "-v", "-i", "1", appName, "-c", "/usr/bin/env && /usr/bin/env >&2")
				Eventually(envCmd).Should(Exit(0))
				output := string(envCmd.Out.Contents())
				stdErr := string(envCmd.Err.Contents())

				Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
				Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

				Expect(string(stdErr)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
				Expect(string(stdErr)).To(MatchRegexp("INSTANCE_INDEX=1"))

				Eventually(func() *Buffer {
					return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
				}).Should(Say("Successful remote access"))

				Eventually(func() string {
					return string(cf.Cf("events", appName).Wait().Out.Contents())
				}).Should(MatchRegexp("audit.app.ssh-authorized"))
			})
		})

		It("can execute a remote command in the container", func() {
			envCmd := cf.Cf("ssh", "-v", appName, "-c", "/usr/bin/env && /usr/bin/env >&2")
			Expect(envCmd.Wait()).To(Exit(0))

			output := string(envCmd.Out.Contents())
			stdErr := string(envCmd.Err.Contents())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Expect(string(stdErr)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(stdErr)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Eventually(func() *Buffer {
				return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
			}).Should(Say("Successful remote access"))
			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("audit.app.ssh-authorized"))
		})

		It("runs an interactive session when no command is provided", func() {
			envCmd := exec.Command("cf", "ssh", "-v", appName)

			stdin, err := envCmd.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			stdout, err := envCmd.StdoutPipe()
			Expect(err).NotTo(HaveOccurred())

			stderr, err := envCmd.StderrPipe()
			Expect(err).NotTo(HaveOccurred())

			err = envCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			_, err = stdin.Write([]byte("/usr/bin/env\n"))
			Expect(err).NotTo(HaveOccurred())

			err = stdin.Close()
			Expect(err).NotTo(HaveOccurred())

			output, err := ioutil.ReadAll(stdout)
			Expect(err).NotTo(HaveOccurred())

			errOutput, err := ioutil.ReadAll(stderr)
			Expect(err).NotTo(HaveOccurred())

			exitErr := envCmd.Wait()
			Expect(exitErr).NotTo(HaveOccurred(), "Failed to run SSH command: %s, %s", output, errOutput)

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Eventually(func() *Buffer {
				return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
			}).Should(Say("Successful remote access"))

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("audit.app.ssh-authorized"))
		})

		It("allows local port forwarding", func() {
			listenCmd := exec.Command("cf", "ssh", "-v", "-L", "127.0.0.1:61007:localhost:8080", appName)

			stdin, err := listenCmd.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			err = listenCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				curl := helpers.Curl(Config, "http://127.0.0.1:61007/").Wait()
				return string(curl.Out.Contents())
			}).Should(ContainSubstring("Catnip?"))

			err = stdin.Close()
			Expect(err).NotTo(HaveOccurred())

			err = listenCmd.Wait()
			Expect(err).NotTo(HaveOccurred())
		})

		It("records successful ssh attempts", func() {
			password := sshAccessCode()

			clientConfig := &ssh.ClientConfig{
				User:            fmt.Sprintf("cf:%s/%d", GuidForAppName(appName), 0),
				Auth:            []ssh.AuthMethod{ssh.Password(password)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}

			client, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).NotTo(HaveOccurred())

			session, err := client.NewSession()
			Expect(err).NotTo(HaveOccurred())

			output, err := session.CombinedOutput("/usr/bin/env")
			Expect(err).NotTo(HaveOccurred())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=0"))

			Eventually(func() *Buffer {
				return logs.Tail(Config.GetUseLogCache(), appName).Wait().Out
			}).Should(Say("Successful remote access"))

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("audit.app.ssh-authorized"))
		})

		It("records failed ssh attempts", func() {
			Eventually(cf.Cf("disable-ssh", appName)).Should(Exit(0))

			password := sshAccessCode()
			clientConfig := &ssh.ClientConfig{
				User:            fmt.Sprintf("cf:%s/%d", GuidForAppName(appName), 0),
				Auth:            []ssh.AuthMethod{ssh.Password(password)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}

			_, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).To(HaveOccurred())

			Eventually(func() string {
				return string(cf.Cf("events", appName).Wait().Out.Contents())
			}).Should(MatchRegexp("audit.app.ssh-unauthorized"))
		})
	})

})

func sshAccessCode() string {
	getCode := cf.Cf("ssh-code")
	Eventually(getCode).Should(Exit(0))
	return strings.TrimSpace(string(getCode.Buffer().Contents()))
}

func sshProxyAddress() string {
	infoCommand := cf.Cf("curl", "/v2/info")
	Expect(infoCommand.Wait()).To(Exit(0))

	type infoResponse struct {
		AppSSHEndpoint string `json:"app_ssh_endpoint"`
	}

	var response infoResponse
	err := json.Unmarshal(infoCommand.Buffer().Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	return response.AppSSHEndpoint
}
