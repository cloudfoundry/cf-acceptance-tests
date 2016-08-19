package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

type Context interface {
	Setup()
	Teardown()

	AdminUserContext() workflowhelpers.UserContext
	RegularUserContext() workflowhelpers.UserContext

	ShortTimeout() time.Duration
	LongTimeout() time.Duration
}

type context struct {
	config Config

	shortTimeout time.Duration
	longTimeout  time.Duration

	organizationName string
	spaceName        string

	quotaDefinitionName string
	quotaDefinitionGUID string

	regularUserUsername string
	regularUserPassword string

	securityGroupName string

	useExistingOrg   bool
	useExistingSpace bool

	originalCfHomeDir string
	currentCfHomeDir  string
}

type QuotaDefinition struct {
	Name string `json:"name"`

	NonBasicServicesAllowed bool `json:"non_basic_services_allowed"`

	TotalServices int `json:"total_services"`
	TotalRoutes   int `json:"total_routes"`

	MemoryLimit int `json:"memory_limit"`
}

func NewContext(config Config, prefix string) Context {
	node := ginkgoconfig.GinkgoConfig.ParallelNode
	timeTag := time.Now().Format("2006_01_02-15h04m05.999s")

	var organizationName string
	var spaceName string
	var useExistingOrg bool
	var useExistingSpace bool

	if config.OrgName != "" {
		useExistingOrg = true
		organizationName = config.OrgName
	} else {
		useExistingOrg = false
		organizationName = fmt.Sprintf("%s-ORG-%d-%s", prefix, node, timeTag)
	}

	if config.SpaceName != "" {
		useExistingSpace = true
		spaceName = config.SpaceName
	} else {
		useExistingSpace = false
		spaceName = fmt.Sprintf("%s-SPACE-%d-%s", prefix, node, timeTag)
	}

	regUserPass := "meow"
	if config.ConfigurableTestPassword != "" {
		regUserPass = config.ConfigurableTestPassword
	}

	return &context{
		config: config,

		shortTimeout: config.ScaledTimeout(1 * time.Minute),
		longTimeout:  config.ScaledTimeout(5 * time.Minute),

		quotaDefinitionName: fmt.Sprintf("%s-QUOTA-%d-%s", prefix, node, timeTag),

		organizationName: organizationName,
		spaceName:        spaceName,

		regularUserUsername: fmt.Sprintf("%s-USER-%d-%s", prefix, node, timeTag),
		regularUserPassword: regUserPass,

		securityGroupName: fmt.Sprintf("%s-SECURITY_GROUP-%d-%s", prefix, node, timeTag),

		useExistingOrg:   useExistingOrg,
		useExistingSpace: useExistingSpace,
	}
}

func (c context) ShortTimeout() time.Duration {
	return c.shortTimeout
}

func (c context) LongTimeout() time.Duration {
	return c.longTimeout
}

func (c *context) Setup() {
	workflowhelpers.AsUser(c.AdminUserContext(), c.shortTimeout, func() {
		EventuallyWithOffset(1, cf.Cf("create-user", c.regularUserUsername, c.regularUserPassword), c.shortTimeout).Should(Exit(0))

		if c.useExistingOrg == false {
			definition := QuotaDefinition{
				Name: c.quotaDefinitionName,

				TotalServices: 100,
				TotalRoutes:   1000,

				MemoryLimit: 10240,

				NonBasicServicesAllowed: true,
			}

			definitionPayload, err := json.Marshal(definition)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			var response workflowhelpers.GenericResource
			workflowhelpers.ApiRequest("POST", "/v2/quota_definitions", &response, c.shortTimeout, string(definitionPayload))

			c.quotaDefinitionGUID = response.Metadata.Guid

			EventuallyWithOffset(1, cf.Cf("create-org", c.organizationName), c.shortTimeout).Should(Exit(0))
			EventuallyWithOffset(1, cf.Cf("set-quota", c.organizationName, c.quotaDefinitionName), c.shortTimeout).Should(Exit(0))
		}

		c.setUpSpaceWithUserAccess(c.RegularUserContext())

		if c.config.CreatePermissiveSecurityGroup {
			c.createPermissiveSecurityGroup()
		}
	})

	c.originalCfHomeDir, c.currentCfHomeDir = workflowhelpers.InitiateUserContext(c.RegularUserContext(), c.shortTimeout)
	workflowhelpers.TargetSpace(c.RegularUserContext(), c.shortTimeout)
}

func (c *context) Teardown() {

	userOrg := c.RegularUserContext().Org

	workflowhelpers.RestoreUserContext(c.RegularUserContext(), c.shortTimeout, c.originalCfHomeDir, c.currentCfHomeDir)

	workflowhelpers.AsUser(c.AdminUserContext(), c.shortTimeout, func() {
		EventuallyWithOffset(1, cf.Cf("delete-user", "-f", c.regularUserUsername), c.longTimeout).Should(Exit(0))

		// delete-space does not provide an org flag, so we must target the Org first
		EventuallyWithOffset(1, cf.Cf("target", "-o", userOrg), c.longTimeout).Should(Exit(0))

		if !c.useExistingSpace {
			EventuallyWithOffset(1, cf.Cf("delete-space", "-f", c.spaceName), c.longTimeout).Should(Exit(0))
		}

		if !c.useExistingOrg {
			EventuallyWithOffset(1, cf.Cf("delete-org", "-f", c.organizationName), c.longTimeout).Should(Exit(0))

			workflowhelpers.ApiRequest(
				"DELETE",
				"/v2/quota_definitions/"+c.quotaDefinitionGUID+"?recursive=true",
				nil,
				c.ShortTimeout(),
			)
		}

		if c.config.CreatePermissiveSecurityGroup {
			EventuallyWithOffset(1, cf.Cf("delete-security-group", "-f", c.securityGroupName), c.shortTimeout).Should(Exit(0))
		}
	})
}

func (c context) AdminUserContext() workflowhelpers.UserContext {
	return workflowhelpers.NewUserContext(
		c.config.ApiEndpoint,
		c.config.AdminUser,
		c.config.AdminPassword,
		"",
		"",
		c.config.SkipSSLValidation,
	)
}

func (c context) RegularUserContext() workflowhelpers.UserContext {
	return workflowhelpers.NewUserContext(
		c.config.ApiEndpoint,
		c.regularUserUsername,
		c.regularUserPassword,
		c.organizationName,
		c.spaceName,
		c.config.SkipSSLValidation,
	)
}

func (c context) setUpSpaceWithUserAccess(uc workflowhelpers.UserContext) {
	if !c.useExistingSpace {
		EventuallyWithOffset(1, cf.Cf("create-space", "-o", uc.Org, uc.Space), c.shortTimeout).Should(Exit(0))
	}
	EventuallyWithOffset(1, cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceManager"), c.shortTimeout).Should(Exit(0))
	EventuallyWithOffset(1, cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceDeveloper"), c.shortTimeout).Should(Exit(0))
	EventuallyWithOffset(1, cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceAuditor"), c.shortTimeout).Should(Exit(0))
}

func (c context) createPermissiveSecurityGroup() {
	rules := []map[string]string{
		map[string]string{
			"destination": "0.0.0.0-255.255.255.255",
			"protocol":    "all",
		},
	}

	rulesFilePath, err := c.writeJSONToTempFile(rules, fmt.Sprintf("%s-rules.json", c.securityGroupName))
	defer os.RemoveAll(rulesFilePath)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	EventuallyWithOffset(1, cf.Cf("create-security-group", c.securityGroupName, rulesFilePath), c.shortTimeout).Should(Exit(0))
	EventuallyWithOffset(1, cf.Cf("bind-security-group", c.securityGroupName, c.organizationName, c.spaceName), c.shortTimeout).Should(Exit(0))
}

func (c context) writeJSONToTempFile(object interface{}, filePrefix string) (filePath string, err error) {
	file, err := ioutil.TempFile("", filePrefix)
	if err != nil {
		return "", err
	}
	defer file.Close()

	filePath = file.Name()
	defer func() {
		if err != nil {
			os.RemoveAll(filePath)
		}
	}()

	bytes, err := json.Marshal(object)
	if err != nil {
		return "", err
	}

	_, err = file.Write(bytes)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
