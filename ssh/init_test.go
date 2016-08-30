package ssh

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	cf_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

var (
	DEFAULT_TIMEOUT      = 30 * time.Second
	CF_PUSH_TIMEOUT      = 2 * time.Minute
	LONG_CURL_TIMEOUT    = 2 * time.Minute
	DEFAULT_MEMORY_LIMIT = "256M"

	context workflowhelpers.SuiteContext
	config  cf_config.Config

	scpPath  string
	sftpPath string
)

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config = cf_config.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	if config.LongCurlTimeout > 0 {
		LONG_CURL_TIMEOUT = config.LongCurlTimeout * time.Second
	}

	context = workflowhelpers.NewContext(config)
	environment := workflowhelpers.NewEnvironment(context)

	type sshPaths struct {
		SCP  string `json:"scp"`
		SFTP string `json:"sftp"`
	}

	var _ = SynchronizedBeforeSuite(func() []byte {
		scp, err := exec.LookPath("scp")
		Expect(err).NotTo(HaveOccurred())

		sftp, err := exec.LookPath("sftp")
		Expect(err).NotTo(HaveOccurred())

		paths, err := json.Marshal(sshPaths{
			SCP:  scp,
			SFTP: sftp,
		})
		Expect(err).NotTo(HaveOccurred())

		return []byte(paths)
	}, func(encodedSSHPaths []byte) {
		var sshPaths sshPaths
		err := json.Unmarshal(encodedSSHPaths, &sshPaths)
		Expect(err).NotTo(HaveOccurred())

		scpPath = sshPaths.SCP
		sftpPath = sshPaths.SFTP

		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "SSH"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func guidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func SshDescribe(description string, callback func()) bool {
	BeforeEach(func() {
		if !config.IncludeSsh {
			Skip(skip_messages.SkipSSHMessage)
		}
	})
	return Describe("[ssh] "+description, callback)
}
