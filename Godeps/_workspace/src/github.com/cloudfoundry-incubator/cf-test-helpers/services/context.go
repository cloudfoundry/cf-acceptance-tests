package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
)

type Context interface {
	Setup()
	Teardown()

	AdminUserContext() cf.UserContext
	RegularUserContext() cf.UserContext

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

	useExistingOrg bool

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
	var useExistingOrg bool

	if config.OrgName != "" {
		useExistingOrg = true
		organizationName = config.OrgName
	} else {
		useExistingOrg = false
		organizationName = fmt.Sprintf("%s-ORG-%d-%s", prefix, node, timeTag)
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
		spaceName:        fmt.Sprintf("%s-SPACE-%d-%s", prefix, node, timeTag),

		regularUserUsername: fmt.Sprintf("%s-USER-%d-%s", prefix, node, timeTag),
		regularUserPassword: regUserPass,

		securityGroupName: fmt.Sprintf("%s-SECURITY_GROUP-%d-%s", prefix, node, timeTag),

		useExistingOrg: useExistingOrg,
	}
}

func (c context) ShortTimeout() time.Duration {
	return c.shortTimeout
}

func (c context) LongTimeout() time.Duration {
	return c.longTimeout
}

func (c *context) Setup() {
	cf.AsUser(c.AdminUserContext(), c.shortTimeout, func() {
		runner.NewCmdRunner(cf.Cf("create-user", c.regularUserUsername, c.regularUserPassword), c.shortTimeout).Run()

		if c.useExistingOrg == false {

			definition := QuotaDefinition{
				Name: c.quotaDefinitionName,

				TotalServices: 100,
				TotalRoutes:   1000,

				MemoryLimit: 10240,

				NonBasicServicesAllowed: true,
			}

			definitionPayload, err := json.Marshal(definition)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			var response cf.GenericResource
			cf.ApiRequest("POST", "/v2/quota_definitions", &response, c.shortTimeout, string(definitionPayload))

			c.quotaDefinitionGUID = response.Metadata.Guid

			runner.NewCmdRunner(cf.Cf("create-org", c.organizationName), c.shortTimeout).Run()
			runner.NewCmdRunner(cf.Cf("set-quota", c.organizationName, c.quotaDefinitionName), c.shortTimeout).Run()
		}

		c.setUpSpaceWithUserAccess(c.RegularUserContext())

		if c.config.CreatePermissiveSecurityGroup {
			c.createPermissiveSecurityGroup()
		}
	})

	c.originalCfHomeDir, c.currentCfHomeDir = cf.InitiateUserContext(c.RegularUserContext(), c.shortTimeout)
	cf.TargetSpace(c.RegularUserContext(), c.shortTimeout)
}

func (c *context) Teardown() {

	userOrg := c.RegularUserContext().Org

	cf.RestoreUserContext(c.RegularUserContext(), c.shortTimeout, c.originalCfHomeDir, c.currentCfHomeDir)

	cf.AsUser(c.AdminUserContext(), c.shortTimeout, func() {
		runner.NewCmdRunner(cf.Cf("delete-user", "-f", c.regularUserUsername), c.longTimeout).Run()

		// delete-space does not provide an org flag, so we must target the Org first
		runner.NewCmdRunner(cf.Cf("target", "-o", userOrg), c.longTimeout).Run()
		runner.NewCmdRunner(cf.Cf("delete-space", "-f", c.spaceName), c.longTimeout).Run()

		if !c.useExistingOrg {
			runner.NewCmdRunner(cf.Cf("delete-org", "-f", c.organizationName), c.longTimeout).Run()

			cf.ApiRequest(
				"DELETE",
				"/v2/quota_definitions/"+c.quotaDefinitionGUID+"?recursive=true",
				nil,
				c.ShortTimeout(),
			)
		}

		if c.config.CreatePermissiveSecurityGroup {
			runner.NewCmdRunner(cf.Cf("delete-security-group", "-f", c.securityGroupName), c.shortTimeout).Run()
		}
	})
}

func (c context) AdminUserContext() cf.UserContext {
	return cf.NewUserContext(
		c.config.ApiEndpoint,
		c.config.AdminUser,
		c.config.AdminPassword,
		"",
		"",
		c.config.SkipSSLValidation,
	)
}

func (c context) RegularUserContext() cf.UserContext {
	return cf.NewUserContext(
		c.config.ApiEndpoint,
		c.regularUserUsername,
		c.regularUserPassword,
		c.organizationName,
		c.spaceName,
		c.config.SkipSSLValidation,
	)
}

func (c context) setUpSpaceWithUserAccess(uc cf.UserContext) {
	runner.NewCmdRunner(cf.Cf("create-space", "-o", uc.Org, uc.Space), c.shortTimeout).Run()
	runner.NewCmdRunner(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceManager"), c.shortTimeout).Run()
	runner.NewCmdRunner(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceDeveloper"), c.shortTimeout).Run()
	runner.NewCmdRunner(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceAuditor"), c.shortTimeout).Run()
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
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	runner.NewCmdRunner(cf.Cf("create-security-group", c.securityGroupName, rulesFilePath), c.shortTimeout).Run()
	runner.NewCmdRunner(cf.Cf("bind-security-group", c.securityGroupName, c.organizationName, c.spaceName), c.shortTimeout).Run()
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
