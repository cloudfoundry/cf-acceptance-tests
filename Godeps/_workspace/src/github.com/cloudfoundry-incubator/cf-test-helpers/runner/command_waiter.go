package runner

import (
	"fmt"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gbytes"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"
)

type cmdWaiter struct {
	session  *gexec.Session
	timeout  time.Duration
	attempts int
	exitCode int
	output   string
}

// NewCmdRunner has default value of exitCode to be 0, and attempts to be 1.
// To change these, use the builder methods WithExitCode and WithAttempts.
func NewCmdWaiter(session *gexec.Session, timeout time.Duration) *cmdWaiter {
	return &cmdWaiter{
		exitCode: 0,
		attempts: 1,
		timeout:  timeout,
		session:  session,
		output:   "",
	}
}

func (c *cmdWaiter) Wait() *gexec.Session {
	cmd := c.session.Command
	cmdString := strings.Join(cmd.Args, " ")
	var exitCode int
	var failureMessage string

	failureMessage = c.wait(cmdString)
	for retries := 0; retries < c.attempts; retries++ {
		exitCode = c.session.ExitCode()

		outputFound := strings.Contains(string(c.session.Buffer().Contents()), c.output)
		if exitCode == c.exitCode && outputFound {
			break
		}

		cmdStarter := NewCommandStarter()
		c.session = cmdStarter.Start(cmd.Args[0], cmd.Args[1:]...)
		failureMessage = c.wait(cmdString)
	}

	Expect(exitCode).To(Equal(c.exitCode), failureMessage)
	Expect(c.session).To(gbytes.Say(c.output))

	return c.session
}

func (c *cmdWaiter) wait(cmdString string) string {
	timer := time.NewTimer(c.timeout)
	var failureMessage string
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
	return failureMessage
}

func (c *cmdWaiter) WithAttempts(attempts int) *cmdWaiter {
	c.attempts = attempts
	return c
}

func (c *cmdWaiter) WithOutput(output string) *cmdWaiter {
	c.output = output
	return c
}

func (c *cmdWaiter) WithExitCode(exitCode int) *cmdWaiter {
	c.exitCode = exitCode
	return c
}
