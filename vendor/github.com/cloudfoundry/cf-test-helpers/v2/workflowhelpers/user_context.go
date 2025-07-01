package workflowhelpers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/commandstarter"
	"github.com/cloudfoundry/cf-test-helpers/v2/internal"
	workflowhelpersinternal "github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers/internal"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

type userValues interface {
	Username() string
	Password() string
	Origin() string
}

type spaceValues interface {
	OrganizationName() string
	SpaceName() string
}

type UserContext struct {
	ApiUrl    string
	TestSpace spaceValues
	TestUser  userValues

	SkipSSLValidation bool
	CommandStarter    internal.Starter
	Timeout           time.Duration

	// the followings are left around for CATS to use
	Username string
	Password string
	Org      string
	Space    string
	Origin   string

	UseClientCredentials bool
}

func cliErrorMessage(session *gexec.Session) string {
	var command string

	if strings.EqualFold(session.Command.Args[1], "auth") {
		command = strings.Join(session.Command.Args[:2], " ")
	} else {
		command = strings.Join(session.Command.Args, " ")
	}

	return fmt.Sprintf("\n>>> [ %s ] exited with an error \n", command)
}

func apiErrorMessage(session *gexec.Session) string {
	apiEndpoint := strings.Join(session.Command.Args, " ")
	stdError := string(session.Err.Contents())

	return fmt.Sprintf("\n>>> [ %s ] exited with an error \n\n%s\n", apiEndpoint, stdError)
}

func NewUserContext(apiUrl string, testUser userValues, testSpace spaceValues, skipSSLValidation bool, timeout time.Duration) UserContext {
	var org, space string
	if testSpace != nil {
		org = testSpace.OrganizationName()
		space = testSpace.SpaceName()
	}

	return UserContext{
		ApiUrl:            apiUrl,
		Username:          testUser.Username(),
		Password:          testUser.Password(),
		Origin:            testUser.Origin(),
		TestSpace:         testSpace,
		TestUser:          testUser,
		Org:               org,
		Space:             space,
		SkipSSLValidation: skipSSLValidation,
		CommandStarter:    commandstarter.NewCommandStarter(),
		Timeout:           timeout,
	}
}

func (uc UserContext) Login() {
	args := []string{"api", uc.ApiUrl}
	if uc.SkipSSLValidation {
		args = append(args, "--skip-ssl-validation")
	}

	session := internal.Cf(uc.CommandStarter, args...).Wait(uc.Timeout)
	gomega.EventuallyWithOffset(1, session, uc.Timeout).Should(gexec.Exit(0), apiErrorMessage(session))

	redactor := internal.NewRedactor(uc.TestUser.Password())
	redactingReporter := internal.NewRedactingReporter(ginkgo.GinkgoWriter, redactor)

	var err error
	if uc.UseClientCredentials {
		err = workflowhelpersinternal.CfClientAuth(uc.CommandStarter, redactingReporter, uc.TestUser.Username(), uc.TestUser.Password(), uc.Timeout)
	} else {
		err = workflowhelpersinternal.CfAuth(uc.CommandStarter, redactingReporter, uc.TestUser.Username(), uc.TestUser.Password(), uc.TestUser.Origin(), uc.Timeout)
	}

	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func (uc UserContext) SetCfHomeDir() (string, string) {
	originalCfHomeDir := os.Getenv("CF_HOME")
	currentCfHomeDir, err := os.MkdirTemp("", fmt.Sprintf("cf_home_%d", ginkgo.GinkgoParallelProcess()))
	if err != nil {
		panic("Error: could not create temporary home directory: " + err.Error())
	}

	err = os.Setenv("CF_HOME", currentCfHomeDir)
	if err != nil {
		panic("Error: could not set 'CF_HOME' env var: " + err.Error())
	}

	return originalCfHomeDir, currentCfHomeDir
}

func (uc UserContext) TargetSpace() {
	if uc.TestSpace != nil && uc.TestSpace.OrganizationName() != "" {
		session := internal.Cf(uc.CommandStarter, "target", "-o", uc.TestSpace.OrganizationName(), "-s", uc.TestSpace.SpaceName())
		gomega.EventuallyWithOffset(1, session, uc.Timeout).Should(gexec.Exit(0), cliErrorMessage(session))
	}
}

func (uc UserContext) AddUserToSpace() {
	username := uc.TestUser.Username()
	orgName := uc.TestSpace.OrganizationName()
	spaceName := uc.TestSpace.SpaceName()

	spaceManager := internal.Cf(uc.CommandStarter, "set-space-role", username, orgName, spaceName, "SpaceManager")
	gomega.EventuallyWithOffset(1, spaceManager, uc.Timeout).Should(gexec.Exit())
	if spaceManager.ExitCode() != 0 {
		gomega.ExpectWithOffset(1, spaceManager.Out).Should(gbytes.Say("not authorized"))
	}

	spaceDeveloper := internal.Cf(uc.CommandStarter, "set-space-role", username, orgName, spaceName, "SpaceDeveloper")
	gomega.EventuallyWithOffset(1, spaceDeveloper, uc.Timeout).Should(gexec.Exit())
	if spaceDeveloper.ExitCode() != 0 {
		gomega.ExpectWithOffset(1, spaceDeveloper.Out).Should(gbytes.Say("not authorized"))
	}

	spaceAuditor := internal.Cf(uc.CommandStarter, "set-space-role", username, orgName, spaceName, "SpaceAuditor")
	gomega.EventuallyWithOffset(1, spaceAuditor, uc.Timeout).Should(gexec.Exit())
	if spaceAuditor.ExitCode() != 0 {
		gomega.ExpectWithOffset(1, spaceAuditor.Out).Should(gbytes.Say("not authorized"))
	}
}

func (uc UserContext) Logout() {
	session := internal.Cf(uc.CommandStarter, "logout")
	gomega.EventuallyWithOffset(1, session, uc.Timeout).Should(gexec.Exit(0), cliErrorMessage(session))
}

func (uc UserContext) UnsetCfHomeDir(originalCfHomeDir, currentCfHomeDir string) {
	err := os.Setenv("CF_HOME", originalCfHomeDir)
	if err != nil {
		panic(err)
	}
	err = os.RemoveAll(currentCfHomeDir)
	if err != nil {
		panic(err)
	}
}
