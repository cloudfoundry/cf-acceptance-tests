package workflowhelpers

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers/internal"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var AsUser = func(userContext UserContext, timeout time.Duration, actions func()) {
	originalCfHomeDir, currentCfHomeDir := InitiateUserContext(userContext, timeout)
	defer func() {
		RestoreUserContext(userContext, timeout, originalCfHomeDir, currentCfHomeDir)
	}()

	TargetSpace(userContext, timeout)

	actions()
}

func InitiateUserContext(userContext UserContext, timeout time.Duration) (originalCfHomeDir, currentCfHomeDir string) {
	originalCfHomeDir = os.Getenv("CF_HOME")
	currentCfHomeDir, err := ioutil.TempDir("", fmt.Sprintf("cf_home_%d", ginkgoconfig.GinkgoConfig.ParallelNode))

	if err != nil {
		panic("Error: could not create temporary home directory: " + err.Error())
	}

	os.Setenv("CF_HOME", currentCfHomeDir)

	cfSetApiArgs := []string{"api", userContext.ApiUrl}
	if userContext.SkipSSLValidation {
		cfSetApiArgs = append(cfSetApiArgs, "--skip-ssl-validation")
	}

	cf.Cf(cfSetApiArgs...).Wait(timeout)

	cmdStarter := commandstarter.NewCommandStarter()
	internal.CfAuth(userContext.Username, userContext.Password, cmdStarter).Wait(timeout)

	return
}

func TargetSpace(userContext UserContext, timeout time.Duration) {
	if userContext.Org != "" {
		if userContext.Space != "" {
			EventuallyWithOffset(1, cf.Cf("target", "-o", userContext.Org, "-s", userContext.Space), timeout).Should(Exit(0))
		} else {
			EventuallyWithOffset(1, cf.Cf("target", "-o", userContext.Org), timeout).Should(Exit(0))
		}
	}
}

func RestoreUserContext(_ UserContext, timeout time.Duration, originalCfHomeDir, currentCfHomeDir string) {
	EventuallyWithOffset(1, cf.Cf("logout"), timeout).Should(Exit(0))
	os.Setenv("CF_HOME", originalCfHomeDir)
	os.RemoveAll(currentCfHomeDir)
}
