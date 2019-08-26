// +build !noInternet,!noDocker

package docker

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = DockerDescribe("Private Docker Registry Application Lifecycle", func() {
	var (
		appName  string
		username string
		password string
	)

	type dockerCreds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type createAppRequest struct {
		Name              string      `json:"name"`
		SpaceGuid         string      `json:"space_guid"`
		DockerImage       string      `json:"docker_image"`
		DockerCredentials dockerCreds `json:"docker_credentials"`
	}

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

		newAppRequest, err := json.Marshal(createAppRequest{
			Name:        appName,
			SpaceGuid:   spaceGuid,
			DockerImage: Config.GetPrivateDockerRegistryImage(),
			DockerCredentials: dockerCreds{
				Username: username,
				Password: password,
			}})

		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("cf", "curl", "-X", "POST", "/v2/apps", "-d", string(newAppRequest))
		cfCurlSession, err := Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		// Redact the docker password from the test logs
		cmd.Args[6] = strings.Replace(cmd.Args[6], `"password":"`+password+`"`, `"password":"***"`, 1)
		reporter := commandreporter.NewCommandReporter()
		reporter.Report(time.Now(), cmd)

		Eventually(cfCurlSession).Should(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	PContext("when the correct username and password are given", func() {
		BeforeEach(func() {
			username = Config.GetPrivateDockerRegistryUsername()
			password = Config.GetPrivateDockerRegistryPassword()
		})

		It("starts the docker app successfully", func() {
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(cf.Cf("map-route", appName, Config.GetAppsDomain(), "--hostname", appName)).Should(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/INSTANCE_INDEX")
			}).Should(Equal("0"))
		})

		It("can run a task", func() {
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			taskName := appName + "-task"
			createCommand := cf.Cf("run-task", appName, "exit 0", "--name", taskName).Wait()
			Expect(createCommand).To(Exit(0))
			Eventually(func() string {
				listCommand := cf.Cf("tasks", appName).Wait()
				Expect(listCommand).To(Exit(0))
				listOutput := string(listCommand.Out.Contents())
				lines := strings.Split(listOutput, "\n")
				if len(lines) != 6 {
					return ""
				}

				fields := strings.Fields(lines[4])
				Expect(fields[1]).To(Equal(taskName))
				return fields[2]
			}).Should(Equal("SUCCEEDED"))
		})
	})
})
