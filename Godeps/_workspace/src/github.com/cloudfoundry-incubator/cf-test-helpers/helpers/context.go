package helpers

import (
	"fmt"
	"time"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
)

const RUNAWAY_QUOTA_MEM_LIMIT = "99999G"

type ConfiguredContext struct {
	config Config

	shortTimeout time.Duration
	longTimeout  time.Duration

	organizationName string
	spaceName        string

	quotaDefinitionName string

	regularUserUsername string
	regularUserPassword string

	isPersistent bool
}

type quotaDefinition struct {
	Name string

	TotalServices string
	TotalRoutes   string
	MemoryLimit   string

	NonBasicServicesAllowed bool
}

func NewContext(config Config) *ConfiguredContext {
	node := ginkgoconfig.GinkgoConfig.ParallelNode
	timeTag := time.Now().Format("2006_01_02-15h04m05.999s")

	regUser := fmt.Sprintf("CATS-USER-%d-%s", node, timeTag)
	regUserPass := "meow"

	if config.UseExistingUser {
		regUser = config.ExistingUser
		regUserPass = config.ExistingUserPassword
	}

	return &ConfiguredContext{
		config: config,

		shortTimeout: config.ScaledTimeout(1 * time.Minute),
		longTimeout:  config.ScaledTimeout(5 * time.Minute),

		quotaDefinitionName: fmt.Sprintf("CATS-QUOTA-%d-%s", node, timeTag),

		organizationName: fmt.Sprintf("CATS-ORG-%d-%s", node, timeTag),
		spaceName:        fmt.Sprintf("CATS-SPACE-%d-%s", node, timeTag),

		regularUserUsername: regUser,
		regularUserPassword: regUserPass,

		isPersistent: false,
	}
}

func NewPersistentAppContext(config Config) *ConfiguredContext {
	baseContext := NewContext(config)

	baseContext.quotaDefinitionName = config.PersistentAppQuotaName
	baseContext.organizationName = config.PersistentAppOrg
	baseContext.spaceName = config.PersistentAppSpace
	baseContext.isPersistent = true

	return baseContext
}

func (context ConfiguredContext) ShortTimeout() time.Duration {
	return context.shortTimeout
}

func (context ConfiguredContext) LongTimeout() time.Duration {
	return context.longTimeout
}

func (context *ConfiguredContext) Setup() {
	cf.AsUser(context.AdminUserContext(), context.shortTimeout, func() {
		definition := quotaDefinition{
			Name: context.quotaDefinitionName,

			TotalServices: "100",
			TotalRoutes:   "1000",
			MemoryLimit:   "10G",

			NonBasicServicesAllowed: true,
		}

		args := []string{
			"create-quota",
			context.quotaDefinitionName,
			"-m", definition.MemoryLimit,
			"-r", definition.TotalRoutes,
			"-s", definition.TotalServices,
		}
		if definition.NonBasicServicesAllowed {
			args = append(args, "--allow-paid-service-plans")
		}

		runner.NewCmdRunner(cf.Cf(args...), context.shortTimeout).Run()

		if !context.config.UseExistingUser {
			createUserCmd := cf.Cf("create-user", context.regularUserUsername, context.regularUserPassword)
			runner.NewCmdRunner(createUserCmd, context.shortTimeout).Run()
			if createUserCmd.ExitCode() != 0 {
				Expect(createUserCmd.Out).To(Say("scim_resource_already_exists"))
			}
		}

		runner.NewCmdRunner(cf.Cf("create-org", context.organizationName), context.shortTimeout).Run()
		runner.NewCmdRunner(cf.Cf("set-quota", context.organizationName, definition.Name), context.shortTimeout).Run()
	})
}

func (context *ConfiguredContext) SetRunawayQuota() {
	cf.AsUser(context.AdminUserContext(), context.shortTimeout, func() {
		runner.NewCmdRunner(cf.Cf("update-quota", context.quotaDefinitionName, "-m", RUNAWAY_QUOTA_MEM_LIMIT, "-i=-1"), context.shortTimeout).Run()
	})
}

func (context *ConfiguredContext) Teardown() {
	cf.AsUser(context.AdminUserContext(), context.shortTimeout, func() {

		if !context.config.ShouldKeepUser {
			runner.NewCmdRunner(cf.Cf("delete-user", "-f", context.regularUserUsername), context.shortTimeout).Run()
		}

		if !context.isPersistent {
			runner.NewCmdRunner(cf.Cf("delete-org", "-f", context.organizationName), context.shortTimeout).Run()
			runner.NewCmdRunner(cf.Cf("delete-quota", "-f", context.quotaDefinitionName), context.shortTimeout).Run()
		}
	})
}

func (context *ConfiguredContext) AdminUserContext() cf.UserContext {
	return cf.NewUserContext(
		context.config.ApiEndpoint,
		context.config.AdminUser,
		context.config.AdminPassword,
		"",
		"",
		context.config.SkipSSLValidation,
	)
}

func (context *ConfiguredContext) RegularUserContext() cf.UserContext {
	return cf.NewUserContext(
		context.config.ApiEndpoint,
		context.regularUserUsername,
		context.regularUserPassword,
		context.organizationName,
		context.spaceName,
		context.config.SkipSSLValidation,
	)
}
