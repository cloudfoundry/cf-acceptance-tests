//go:build !noInternet && !noDocker
// +build !noInternet,!noDocker

package docker

import (
	"os"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = DockerDescribe("Private Docker Registry Application Lifecycle", func() {
	var (
		appName    string
		username   string
		password   string
		repository string
	)

	BeforeEach(func() {
		if !Config.GetIncludePrivateDockerRegistry() {
			Skip(skip_messages.SkipPrivateDockerRegistryMessage)
		}
	})

	JustBeforeEach(func() {
		spaceName := TestSetup.RegularUserContext().Space
		session := cf.Cf("space", spaceName, "--guid")
		Eventually(session).Should(Exit(0))
		spaceGuid := string(session.Out.Contents())
		spaceGuid = strings.TrimSpace(spaceGuid)
		appName = random_name.CATSRandomName("APP")

		os.Setenv("CF_DOCKER_PASSWORD", password)
		Eventually(cf.Cf("push", appName,
			"--docker-image", repository,
			"--docker-username", username)).Should(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Context("when the correct username and password are given", func() {
		BeforeEach(func() {
			username = Config.GetPrivateDockerRegistryUsername()
			password = Config.GetPrivateDockerRegistryPassword()
			repository = Config.GetPrivateDockerRegistryImage()
		})

		It("starts the docker app successfully", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/INSTANCE_INDEX")
			}).Should(Equal("0"))
		})

		It("can run a task", func() {
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			taskName := appName + "-task"
			createCommand := cf.Cf("run-task", appName, "--command", "exit 0", "--name", taskName).Wait()
			Expect(createCommand).To(Exit(0))
			Eventually(func() string {
				listCommand := cf.Cf("tasks", appName).Wait()
				Expect(listCommand).To(Exit(0))
				listOutput := string(listCommand.Out.Contents())
				lines := strings.Split(listOutput, "\n")
				if len(lines) != 5 {
					return ""
				}

				fields := strings.Fields(lines[3])
				Expect(fields[1]).To(Equal(taskName))
				return fields[2]
			}).Should(Equal("SUCCEEDED"))
		})
	})
})
