package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
)

type ConfiguredContext struct {
	config Config

	organizationName string

	quotaDefinitionName string
	quotaDefinitionGUID string

	regularUserUsername string
	regularUserPassword string
}

type quotaDefinition struct {
	Name string `json:"name"`

	NonBasicServicesAllowed bool `json:"non_basic_services_allowed"`

	TotalServices int `json:"total_services"`
	TotalRoutes   int `json:"total_routes"`

	MemoryLimit int `json:"memory_limit"`
}

func NewContext(config Config) *ConfiguredContext {
	return &ConfiguredContext{
		config: config,

		quotaDefinitionName: fmt.Sprintf(time.Now().Format("CATS-QUOTA-%d-2006_01_02-15h04m"), ginkgoconfig.GinkgoConfig.ParallelNode),

		organizationName: fmt.Sprintf(time.Now().Format("CATS-%d-2006_01_02-15h04m"), ginkgoconfig.GinkgoConfig.ParallelNode),

		regularUserUsername: fmt.Sprintf(time.Now().Format("CATS-USER-%d-2006_01_02-15h04m"), ginkgoconfig.GinkgoConfig.ParallelNode),
		regularUserPassword: "password",
	}
}

func (context *ConfiguredContext) Setup() {
	cf.AsUser(context.AdminUserContext(), func() {
		definition := quotaDefinition{
			Name: context.quotaDefinitionName,

			TotalServices: 100,
			TotalRoutes:   1000,

			MemoryLimit: 10240,

			NonBasicServicesAllowed: true,
		}

		definitionPayload, err := json.Marshal(definition)
		Expect(err).ToNot(HaveOccurred())

		var response cf.GenericResource

		cf.ApiRequest("POST", "/v2/quota_definitions", &response, string(definitionPayload))

		context.quotaDefinitionGUID = response.Metadata.Guid
		fmt.Printf("QuotaDefinition Response: %#v\n", response)
		println("GUID", context.quotaDefinitionGUID)

		Expect(cf.Cf("create-user", context.regularUserUsername, context.regularUserPassword)).To(SayBranches(
			cmdtest.ExpectBranch{"OK", func() {}},
			cmdtest.ExpectBranch{"scim_resource_already_exists", func() {}},
		))

		Expect(cf.Cf("create-org", context.organizationName)).To(ExitWith(0))
		Expect(cf.Cf("set-quota", context.organizationName, definition.Name)).To(ExitWith(0))
	})
}

func (context *ConfiguredContext) Teardown() {
	cf.AsUser(context.AdminUserContext(), func() {
		Expect(cf.Cf("delete-org", "-f", context.organizationName)).To(ExitWith(0))

		Expect(cf.Cf("delete-user", "-f", context.regularUserUsername)).To(ExitWith(0))

		cf.ApiRequest(
			"DELETE",
			"/v2/quota_definitions/"+context.quotaDefinitionGUID+"?recursive=true",
			nil,
		)
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
		fmt.Sprintf("CATS-space-%d", ginkgoconfig.GinkgoConfig.ParallelNode),
		context.config.SkipSSLValidation,
	)
}
