package cf_test

import (
	"bytes"
	"os/exec"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gbytes"
)

var _ = Describe("CfAuth", func() {
	var callerOutuput *bytes.Buffer
	var password string

	BeforeEach(func() {
		callerOutuput = bytes.NewBuffer([]byte{})
		password = "superSecretPassword"

		GinkgoWriter = callerOutuput
	})

	It("runs the cf auth command", func() {
		user := "myUser"

		runner.CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
			Expect(cmd.Path).To(Equal(exec.Command("cf").Path))
			Expect(cmd.Args).To(Equal([]string{
				"cf", "auth", user, password,
			}))

			return exec.Command("bash", "-c", "echo \"Authenticating...\nOK\"")
		}

		Eventually(cf.CfAuth(user, password)).Should(gbytes.Say("Authenticating...\nOK"))
	})

	It("does not expose the password", func() {
		user := "myUser"

		runner.CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
			Expect(cmd.Path).To(Equal(exec.Command("cf").Path))
			Expect(cmd.Args).To(Equal([]string{
				"cf", "auth", user, password,
			}))

			return exec.Command("bash", "-c", "echo \"Authenticating...\nOK\"")
		}

		cf.CfAuth(user, password).Wait()
		Expect(callerOutuput.String()).NotTo(ContainSubstring(password))
		Expect(callerOutuput.String()).To(ContainSubstring("REDACTED"))
	})
})
