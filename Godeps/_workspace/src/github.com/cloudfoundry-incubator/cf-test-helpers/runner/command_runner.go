package runner

import (
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"
)

type cmdRunner struct {
	waiter *cmdWaiter
}

// NewCmdRunner has default value of exitCode to be 0, and attempts to be 1.
// To change these, use the builder methods WithExitCode and WithAttempts.
func NewCmdRunner(session *gexec.Session, timeout time.Duration) *cmdRunner {
	waiter := &cmdWaiter{
		exitCode: 0,
		attempts: 1,
		timeout:  timeout,
		session:  session,
		output:   "",
	}

	return &cmdRunner{
		waiter: waiter,
	}
}

func (c *cmdRunner) Run() *gexec.Session {
	return c.waiter.Wait()
}

func (c *cmdRunner) WithAttempts(attempts int) *cmdRunner {
	c.waiter.attempts = attempts
	return c
}

func (c *cmdRunner) WithOutput(output string) *cmdRunner {
	c.waiter.output = output
	return c
}

func (c *cmdRunner) WithExitCode(exitCode int) *cmdRunner {
	c.waiter.exitCode = exitCode
	return c
}
