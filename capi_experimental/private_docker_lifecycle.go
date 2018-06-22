// +build !noInternet,!noDocker

package capi_experimental

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = CapiExperimentalDescribe("Private Docker Registry Application Lifecycle", func() {
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
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Context("when an incorrect username and password are given", func() {
		BeforeEach(func() {
			username = Config.GetPrivateDockerRegistryUsername() + "wrong"
			password = Config.GetPrivateDockerRegistryPassword() + "wrong"
		})

		It("fails to start the docker app since the credentials are invalid", func() {
			session := cf.Cf("start", appName)
			Eventually(session, Config.CfPushTimeoutDuration()).Should(gbytes.Say("(invalid username/password|[Uu]nauthorized)"))
			Eventually(session, Config.CfPushTimeoutDuration()).Should(Exit(1))
		})
	})
})
