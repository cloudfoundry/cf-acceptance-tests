package ssh

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/kr/pty"
	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe(deaUnsupportedTag+"SSH", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
		Eventually(cf.Cf(
			"push", appName,
			"-p", assets.NewAssets().Dora,
			"--no-start",
			"-b", "ruby_buildpack",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-d", config.AppsDomain,
			"-i", "2"),
			DEFAULT_TIMEOUT,
		).Should(Exit(0))

		app_helpers.SetBackend(appName)

		enableSSH(appName)

		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(func() string {
			return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
		}, DEFAULT_TIMEOUT).Should(Equal("1"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	Describe("ssh", func() {
		It("can execute a remote command in the container", func() {
			envCmd := cf.Cf("ssh", "-i", "1", appName, "-c", "/usr/bin/env")
			Expect(envCmd.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			output := string(envCmd.Buffer().Contents())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

			Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Say("Successful remote access"))
			Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-authorized"))
		})

		It("runs an interactive session when no command is provided", func() {
			envCmd := exec.Command("cf", "ssh", "-i", "1", appName)

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
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

			Eventually(cf.Cf("logs", appName, "--recent"), DEFAULT_TIMEOUT).Should(Say("Successful remote access"))
			Eventually(cf.Cf("events", appName), DEFAULT_TIMEOUT).Should(Say("audit.app.ssh-authorized"))
		})

		It("allows local port forwarding", func() {
			listenCmd := exec.Command("cf", "ssh", "-i", "1", "-L", "127.0.0.1:37001:localhost:8080", appName)

			stdin, err := listenCmd.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			err = listenCmd.Start()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				stdout := &bytes.Buffer{}
				curlCmd := exec.Command("curl", "http://127.0.0.1:37001/")
				curlCmd.Stdout = stdout
				curlCmd.Run()
				return stdout.String()
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora"))

			err = stdin.Close()
			Expect(err).NotTo(HaveOccurred())

			err = listenCmd.Wait()
			Expect(err).NotTo(HaveOccurred())
		})

		It("can be ssh'ed to and records its success", func() {
			password := sshAccessCode()

			clientConfig := &ssh.ClientConfig{
				User: fmt.Sprintf("cf:%s/%d", guidForAppName(appName), 1),
				Auth: []ssh.AuthMethod{ssh.Password(password)},
			}

			client, err := ssh.Dial("tcp", sshProxyAddress(), clientConfig)
			Expect(err).NotTo(HaveOccurred())

			session, err := client.NewSession()
			Expect(err).NotTo(HaveOccurred())

			output, err := session.Output("/usr/bin/env")
			Expect(err).NotTo(HaveOccurred())

			Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`VCAP_APPLICATION=.*"application_name":"%s"`, appName)))
			Expect(string(output)).To(MatchRegexp("INSTANCE_INDEX=1"))

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

	Describe("scp", func() {
		var (
			sourceDir, targetDir             string
			generatedFile, generatedFileName string
			generatedFileInfo                os.FileInfo
			err                              error
		)

		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())

			sourceDir, err = ioutil.TempDir("", "scp-source")
			Expect(err).NotTo(HaveOccurred())

			fileContents := make([]byte, 1024)
			b, err := rand.Read(fileContents)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).To(Equal(len(fileContents)))

			generatedFileName = "binary.dat"
			generatedFile = filepath.Join(sourceDir, generatedFileName)

			err = ioutil.WriteFile(generatedFile, fileContents, 0664)
			Expect(err).NotTo(HaveOccurred())

			generatedFileInfo, err = os.Stat(generatedFile)
			Expect(err).NotTo(HaveOccurred())

			targetDir, err = ioutil.TempDir("", "scp-target")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
			}, DEFAULT_TIMEOUT).Should(Equal("0"))
		})

		runScp := func(src, dest string) {
			_, sshPort, err := net.SplitHostPort(sshProxyAddress())
			Expect(err).NotTo(HaveOccurred())

			ptyMaster, ptySlave, err := pty.Open()
			Expect(err).NotTo(HaveOccurred())
			defer ptyMaster.Close()

			password := sshAccessCode() + "\n"

			cmd := exec.Command(scpPath,
				"-r",
				"-P", sshPort,
				fmt.Sprintf("-oUser=cf:%s/0", guidForAppName(appName)),
				"-oUserKnownHostsFile=/dev/null",
				"-oStrictHostKeyChecking=no",
				src,
				dest,
			)

			cmd.Stdin = ptySlave
			cmd.Stdout = ptySlave
			cmd.Stderr = ptySlave

			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setctty: true,
				Setsid:  true,
			}

			sayCommandRun(cmd)
			session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// Close our open reference to ptySlave so that PTY Master recieves EOF
			ptySlave.Close()

			sendPassword(ptyMaster, password)

			done := make(chan struct{})
			go func() {
				io.Copy(GinkgoWriter, ptyMaster)
				close(done)
			}()

			Eventually(done, DEFAULT_TIMEOUT).Should(BeClosed())
			Eventually(session, DEFAULT_TIMEOUT).Should(Exit(0))
		}

		It("can send and receive files over scp", func() {
			sshHost, _, err := net.SplitHostPort(sshProxyAddress())
			Expect(err).NotTo(HaveOccurred())

			runScp(sourceDir, fmt.Sprintf("%s:/home/vcap", sshHost))
			runScp(fmt.Sprintf("%s:/home/vcap/%s", sshHost, filepath.Base(sourceDir)), targetDir)

			compareDir(sourceDir, filepath.Join(targetDir, filepath.Base(sourceDir)))
		})
	})

	Describe("sftp", func() {
		var (
			sourceDir, targetDir             string
			generatedFile, generatedFileName string
			generatedFileInfo                os.FileInfo
			err                              error
		)

		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())

			sourceDir, err = ioutil.TempDir("", "sftp-source")
			Expect(err).NotTo(HaveOccurred())

			fileContents := make([]byte, 1024)
			b, err := rand.Read(fileContents)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).To(Equal(len(fileContents)))

			generatedFileName = "binary.dat"
			generatedFile = filepath.Join(sourceDir, generatedFileName)

			err = ioutil.WriteFile(generatedFile, fileContents, 0664)
			Expect(err).NotTo(HaveOccurred())

			generatedFileInfo, err = os.Stat(generatedFile)
			Expect(err).NotTo(HaveOccurred())

			targetDir, err = ioutil.TempDir("", "sftp-target")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
			}, DEFAULT_TIMEOUT).Should(Equal("0"))
		})

		runSftp := func(stdin io.Reader) *Buffer {
			sshHost, sshPort, err := net.SplitHostPort(sshProxyAddress())
			Expect(err).NotTo(HaveOccurred())

			ptyMaster, ptySlave, err := pty.Open()
			Expect(err).NotTo(HaveOccurred())
			defer ptyMaster.Close()

			password := sshAccessCode() + "\n"

			cmd := exec.Command(
				sftpPath,
				"-P", sshPort,
				"-oUserKnownHostsFile=/dev/null",
				"-oStrictHostKeyChecking=no",
				fmt.Sprintf("cf:%s/0@%s", guidForAppName(appName), sshHost),
			)

			cmd.Stdin = ptySlave
			cmd.Stdout = ptySlave
			cmd.Stderr = ptySlave

			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setctty: true,
				Setsid:  true,
			}

			sayCommandRun(cmd)
			session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// Close our open reference to ptySlave so that PTY Master recieves EOF
			ptySlave.Close()

			sendPassword(ptyMaster, password)

			done := make(chan struct{})
			go func() {
				io.Copy(GinkgoWriter, ptyMaster)
				close(done)
			}()

			go func() {
				io.Copy(ptyMaster, stdin)
				ptyMaster.Write([]byte("exit\n"))
			}()

			Eventually(done, DEFAULT_TIMEOUT).Should(BeClosed())
			Eventually(session, DEFAULT_TIMEOUT).Should(Exit(0))

			return session.Buffer()
		}

		It("defaults to $HOME as the remote working directory", func() {
			output := runSftp(strings.NewReader("pwd\n"))
			Eventually(output, DEFAULT_TIMEOUT).Should(Say("working directory: /home/vcap"))
		})

		It("can send and receive files over sftp", func() {
			input := &bytes.Buffer{}
			input.WriteString("mkdir files\n")
			input.WriteString("cd files\n")
			input.WriteString("lcd " + sourceDir + "\n")
			input.WriteString("put " + generatedFileName + "\n")
			input.WriteString("lcd " + targetDir + "\n")
			input.WriteString("get " + generatedFileName + "\n")

			runSftp(input)

			compareDir(sourceDir, targetDir)
		})
	})
})

func sendPassword(pty *os.File, password string) {
	passwordPrompt := []byte("password: ")

	b := make([]byte, 1)
	buf := []byte{}
	done := make(chan struct{})

	go func() {
		defer GinkgoRecover()
		for {
			n, err := pty.Read(b)
			Expect(n).To(Equal(1))
			Expect(err).NotTo(HaveOccurred())
			buf = append(buf, b[0])
			if bytes.HasSuffix(buf, passwordPrompt) {
				break
			}
		}
		n, err := pty.Write([]byte(password))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(len(password)))

		close(done)
	}()

	Eventually(done, DEFAULT_TIMEOUT).Should(BeClosed())
}

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

func compareDir(actualDir, expectedDir string) {
	actualDirInfo, err := os.Stat(actualDir)
	Expect(err).NotTo(HaveOccurred())

	expectedDirInfo, err := os.Stat(expectedDir)
	Expect(err).NotTo(HaveOccurred())

	Expect(actualDirInfo.Mode()).To(Equal(expectedDirInfo.Mode()))

	actualFiles, err := ioutil.ReadDir(actualDir)
	Expect(err).NotTo(HaveOccurred())

	expectedFiles, err := ioutil.ReadDir(actualDir)
	Expect(err).NotTo(HaveOccurred())

	Expect(len(actualFiles)).To(Equal(len(expectedFiles)))
	for i, actualFile := range actualFiles {
		expectedFile := expectedFiles[i]
		if actualFile.IsDir() {
			compareDir(filepath.Join(actualDir, actualFile.Name()), filepath.Join(expectedDir, expectedFile.Name()))
		} else {
			compareFile(filepath.Join(actualDir, actualFile.Name()), filepath.Join(expectedDir, expectedFile.Name()))
		}
	}
}

func compareFile(actualFile, expectedFile string) {
	actualFileInfo, err := os.Stat(actualFile)
	Expect(err).NotTo(HaveOccurred())

	expectedFileInfo, err := os.Stat(expectedFile)
	Expect(err).NotTo(HaveOccurred())

	Expect(actualFileInfo.Mode()).To(Equal(expectedFileInfo.Mode()))
	Expect(actualFileInfo.Size()).To(Equal(expectedFileInfo.Size()))

	actualContents, err := ioutil.ReadFile(actualFile)
	Expect(err).NotTo(HaveOccurred())

	expectedContents, err := ioutil.ReadFile(expectedFile)
	Expect(err).NotTo(HaveOccurred())

	Expect(actualContents).To(Equal(expectedContents))
}
