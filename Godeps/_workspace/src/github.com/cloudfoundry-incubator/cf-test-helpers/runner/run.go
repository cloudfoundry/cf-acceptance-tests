package runner

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const timeFormat = "2006-01-02 15:04:05.00 (MST)"

var CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
var SkipSSLValidation bool

func Run(executable string, args ...string) *gexec.Session {
	cmd := exec.Command(executable, args...)

	return innerRun(cmd)
}

func innerRun(cmd *exec.Cmd) *gexec.Session {
	sayCommandWillRun(time.Now(), cmd)

	sess, err := gexec.Start(CommandInterceptor(cmd), ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return sess
}

func Curl(args ...string) *gexec.Session {
	curlArgs := append([]string{"-s"}, args...)
	if SkipSSLValidation {
		curlArgs = append([]string{"-k"}, curlArgs...)
	}
	return Run("curl", curlArgs...)
}

func sayCommandWillRun(startTime time.Time, cmd *exec.Cmd) {
	startColor := ""
	endColor := ""
	if !config.DefaultReporterConfig.NoColor {
		startColor = "\x1b[32m"
		endColor = "\x1b[0m"
	}
	fmt.Fprintf(ginkgo.GinkgoWriter, "\n%s[%s]> %s %s\n", startColor, startTime.UTC().Format(timeFormat), strings.Join(cmd.Args, " "), endColor)
}

type cmdRunner struct {
	session  *gexec.Session
	timeout  time.Duration
	attempts int
	exitCode int
	output   string
}

// NewCmdRunner has default value of exitCode to be 0, and attempts to be 1.
// To change these, use the builder methods WithExitCode and WithAttempts.
func NewCmdRunner(session *gexec.Session, timeout time.Duration) *cmdRunner {
	return &cmdRunner{
		exitCode: 0,
		attempts: 1,
		timeout:  timeout,
		session:  session,
		output:   "",
	}
}

func (c *cmdRunner) Run() *gexec.Session {
	cmd := c.session.Command
	cmdString := strings.Join(cmd.Args, " ")
	var exitCode int
	var failureMessage string

	for i := 0; i < c.attempts; i++ {

		// The first time through this loop we use the command that was provided,
		// which is already running.
		// After that we must explicitly start the command
		if i > 0 {
			newCmd := exec.Command(cmd.Args[0], cmd.Args[1:]...)
			c.session = innerRun(newCmd)
		}

		timer := time.NewTimer(c.timeout)

		select {
		case <-timer.C:
			failureMessage = fmt.Sprintf(
				"Timed out executing command (%v):\nCommand: %s\n\n[stdout]:\n%s\n\n[stderr]:\n%s",
				c.timeout.String(),
				cmdString,
				string(c.session.Out.Contents()),
				string(c.session.Err.Contents()))
		case <-c.session.Exited:
			// immediate kill the timer goroutine
			timer.Stop()

			// command may not have failed, but pre-construct failure message for final exit code expectation
			failureMessage = fmt.Sprintf(
				"Failed executing command (exit %d):\nCommand: %s\n\n[stdout]:\n%s\n\n[stderr]:\n%s",
				c.session.ExitCode(),
				cmdString,
				string(c.session.Out.Contents()),
				string(c.session.Err.Contents()))
		}
		exitCode = c.session.ExitCode()

		outputFound := strings.Contains(string(c.session.Buffer().Contents()), c.output)
		if exitCode == c.exitCode && outputFound {
			break
		}
	}

	Expect(exitCode).To(Equal(c.exitCode), failureMessage)
	Expect(c.session).To(gbytes.Say(c.output))

	return c.session
}

func (c *cmdRunner) WithOutput(output string) *cmdRunner {
	c.output = output
	return c
}

func (c *cmdRunner) WithAttempts(attempts int) *cmdRunner {
	c.attempts = attempts
	return c
}

func (c *cmdRunner) WithExitCode(exitCode int) *cmdRunner {
	c.exitCode = exitCode
	return c
}
