package helpers

import (
	"fmt"
	"time"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/vito/cmdtest/matchers"
)

type ConfiguredContext struct {
	config Config

	organizationName string
}

func NewContext(config Config) ConfiguredContext {
	return ConfiguredContext{
		config: config,

		organizationName: fmt.Sprintf(time.Now().Format("CATS-%d-2006_01_02-15h04m"), ginkgoconfig.GinkgoConfig.ParallelNode),
	}
}

func (context ConfiguredContext) Setup() {
	cf.AsUser(context.AdminUserContext(), func() {
		Expect(cf.Cf("create-org", context.organizationName)).To(ExitWith(0))
		Expect(cf.Cf("set-quota", context.organizationName, "runaway")).To(ExitWith(0))
	})
}

func (context ConfiguredContext) Teardown() {
	cf.AsUser(context.AdminUserContext(), func() {
		Expect(cf.Cf("delete-org", "-f", context.organizationName)).To(ExitWith(0))
	})
}

func (context ConfiguredContext) AdminUserContext() cf.UserContext {
	return cf.NewUserContext(
		context.config.ApiEndpoint,
		context.config.AdminUser,
		context.config.AdminPassword,
		"",
		"",
		context.config.SkipSSLValidation,
	)
}

func (context ConfiguredContext) RegularUserContext() cf.UserContext {
	return cf.NewUserContext(
		context.config.ApiEndpoint,
		fmt.Sprintf("CATS-user-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		fmt.Sprintf("CATS-user-pass-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		context.organizationName,
		fmt.Sprintf("CATS-space-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		context.config.SkipSSLValidation,
	)
}
