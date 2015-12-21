package runner_test

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const cmdTimeout = 30 * time.Second

var _ = Describe("Run", func() {
	It("runs the given command in a cmdtest Session", func() {
		session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 42").Wait(cmdTimeout)
		Expect(session).To(Exit(42))
		Expect(session.Out).To(Say("hi out"))
		Expect(session.Err).To(Say("hi err"))
	})
})

var _ = Describe("Curl", func() {
	It("outputs the body of the given URL", func() {
		session := runner.Curl("-I", "http://example.com").Wait(cmdTimeout)
		Expect(session).To(Exit(0))
		Expect(session.Out).To(Say("HTTP/1.1 200 OK"))
	})
})

var _ = Describe("cmdRunner", func() {

	Describe("Run with defaults", func() {
		It("does nothing when the command succeeds before the timeout", func() {
			failures := InterceptGomegaFailures(func() {
				session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 0")
				runner.NewCmdRunner(session, cmdTimeout).Run()
			})
			Expect(failures).To(BeEmpty())
		})

		It("expects the command not to fail", func() {
			failures := InterceptGomegaFailures(func() {
				session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 42")
				runner.NewCmdRunner(session, cmdTimeout).Run()
			})
			Expect(failures[0]).To(MatchRegexp(
				"Failed executing command \\(exit 42\\):\nCommand: %s\n\n\\[stdout\\]:\n%s\n\n\\[stderr\\]:\n%s",
				"bash -c echo hi out; echo hi err 1>&2; exit 42",
				"hi out\n",
				"hi err\n",
			))
		})

		It("expects the command not to time out", func() {
			failures := InterceptGomegaFailures(func() {
				session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; sleep 1")
				runner.NewCmdRunner(session, 100*time.Millisecond).Run()
			})
			Expect(failures[0]).To(MatchRegexp(
				"Timed out executing command \\(100ms\\):\nCommand: %s\n\n\\[stdout\\]:\n%s\n\n\\[stderr\\]:\n%s",
				"bash -c echo hi out; echo hi err 1>&2; sleep 1",
				"hi out\n",
				"hi err\n",
			))
		})

		Describe("WithExitCode", func() {
			It("expects exit code", func() {
				failures := InterceptGomegaFailures(func() {
					session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 42")
					runner.NewCmdRunner(session, cmdTimeout).WithExitCode(42).Run()
				})
				Expect(failures).To(HaveLen(0))
			})
		})

		Describe("WithOutput", func() {
			It("expects output", func() {
				failures := InterceptGomegaFailures(func() {
					session := runner.Run("bash", "-c", "echo hi out; echo hi err 1>&2; exit 0")
					runner.NewCmdRunner(session, cmdTimeout).WithOutput("hi out").Run()
				})
				Expect(failures).To(HaveLen(0))
			})
		})

		Describe("WithAttempts", func() {
			It("retries", func() {
				f, err := ioutil.TempFile("", "tmpFile")
				Expect(err).NotTo(HaveOccurred())
				defer f.Close()
				filepath := f.Name()
				f.WriteString("0")

				attempts := 3
				//reads from file and increments contents by one; exits non-zero until final attempt
				command := fmt.Sprintf(
					"cur_val=$(( $(cat %[1]s ) + 1)); echo $cur_val > %[1]s; exit $(( %[2]d - cur_val ))",
					filepath,
					attempts,
				)

				failures := InterceptGomegaFailures(func() {
					session := runner.Run("bash", "-c", command)
					runner.NewCmdRunner(session, 1*time.Second).WithAttempts(attempts).Run()
				})

				Expect(failures).To(HaveLen(0))

				b, err := ioutil.ReadFile(filepath)
				Expect(err).NotTo(HaveOccurred())

				fileContents := strings.TrimSpace(string(b))
				actualAttempts, err := strconv.Atoi(fileContents)
				Expect(err).NotTo(HaveOccurred())

				Expect(actualAttempts).To(Equal(attempts))
			})
		})
	})
})
