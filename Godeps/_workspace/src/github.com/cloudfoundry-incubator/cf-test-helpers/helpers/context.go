package helpers

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const CF_API_TIMEOUT = 30 * time.Second

type ConfiguredContext struct {
	config Config

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

	return &ConfiguredContext{
		config: config,

		quotaDefinitionName: fmt.Sprintf("CATS-QUOTA-%d-%s", node, timeTag),

		organizationName: fmt.Sprintf("CATS-ORG-%d-%s", node, timeTag),
		spaceName:        fmt.Sprintf("CATS-SPACE-%d-%s", node, timeTag),

		regularUserUsername: fmt.Sprintf("CATS-USER-%d-%s", node, timeTag),
		regularUserPassword: "meow",

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

func (context *ConfiguredContext) Setup() {
	cf.AsUser(context.AdminUserContext(), func() {
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

		Expect(cf.Cf(args...).Wait(CF_API_TIMEOUT)).To(Exit(0))

		createUserSession := cf.Cf("create-user", context.regularUserUsername, context.regularUserPassword)
		createUserSession.Wait(CF_API_TIMEOUT)
		if createUserSession.ExitCode() != 0 {
			Expect(createUserSession.Out).To(Say("scim_resource_already_exists"))
		}

		Expect(cf.Cf("create-org", context.organizationName).Wait(CF_API_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("set-quota", context.organizationName, definition.Name).Wait(CF_API_TIMEOUT)).To(Exit(0))
	})
}

func (context *ConfiguredContext) Teardown() {
	cf.AsUser(context.AdminUserContext(), func() {
		Expect(cf.Cf("delete-user", "-f", context.regularUserUsername).Wait(CF_API_TIMEOUT)).To(Exit(0))

		if !context.isPersistent {
			Expect(cf.Cf("delete-org", "-f", context.organizationName).Wait(CF_API_TIMEOUT)).To(Exit(0))

			Expect(cf.Cf("delete-quota", "-f", context.quotaDefinitionName).Wait(CF_API_TIMEOUT)).To(Exit(0))
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
