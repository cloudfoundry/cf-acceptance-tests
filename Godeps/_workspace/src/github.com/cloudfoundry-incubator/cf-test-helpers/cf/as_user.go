package cf

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	ginkgoconfig "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo/config"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/runner"
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

	runner.NewCmdWaiter(Cf(cfSetApiArgs...), timeout).Wait()

	runner.NewCmdWaiter(CfAuth(userContext.Username, userContext.Password), timeout).Wait()

	return
}

func TargetSpace(userContext UserContext, timeout time.Duration) {
	if userContext.Org != "" {
		if userContext.Space != "" {
			runner.NewCmdWaiter(Cf("target", "-o", userContext.Org, "-s", userContext.Space), timeout).Wait()
		} else {
			runner.NewCmdWaiter(Cf("target", "-o", userContext.Org), timeout).Wait()
		}
	}
}

func RestoreUserContext(_ UserContext, timeout time.Duration, originalCfHomeDir, currentCfHomeDir string) {
	runner.NewCmdWaiter(Cf("logout"), timeout).Wait()
	os.Setenv("CF_HOME", originalCfHomeDir)
	os.RemoveAll(currentCfHomeDir)
}
