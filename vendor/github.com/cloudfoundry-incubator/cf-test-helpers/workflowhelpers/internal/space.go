package internal

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/commandstarter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/internal"
)

type TestSpace struct {
	QuotaDefinitionName                  string
	organizationName                     string
	spaceName                            string
	isPersistent                         bool
	QuotaDefinitionTotalMemoryLimit      string
	QuotaDefinitionInstanceMemoryLimit   string
	QuotaDefinitionRoutesLimit           string
	QuotaDefinitionAppInstanceLimit      string
	QuotaDefinitionServiceInstanceLimit  string
	QuotaDefinitionAllowPaidServicesFlag string
	CommandStarter                       internal.Starter
	Timeout                              time.Duration
}

type spaceConfig interface {
	GetScaledTimeout(time.Duration) time.Duration
	GetPersistentAppSpace() string
	GetPersistentAppOrg() string
	GetPersistentAppQuotaName() string
	GetNamePrefix() string
}

func NewRegularTestSpace(cfg spaceConfig, quotaLimit string) *TestSpace {
	return NewBaseTestSpace(
		generator.PrefixedRandomName(cfg.GetNamePrefix(), "SPACE"),
		generator.PrefixedRandomName(cfg.GetNamePrefix(), "ORG"),
		generator.PrefixedRandomName(cfg.GetNamePrefix(), "QUOTA"),
		quotaLimit,
		false,
		cfg.GetScaledTimeout(1*time.Minute),
		commandstarter.NewCommandStarter(),
	)
}

func NewPersistentAppTestSpace(cfg spaceConfig) *TestSpace {
	baseTestSpace := NewBaseTestSpace(
		cfg.GetPersistentAppSpace(),
		cfg.GetPersistentAppOrg(),
		cfg.GetPersistentAppQuotaName(),
		"10G",
		true,
		cfg.GetScaledTimeout(1*time.Minute),
		commandstarter.NewCommandStarter(),
	)
	return baseTestSpace
}

func NewBaseTestSpace(spaceName, organizationName, quotaDefinitionName, quotaDefinitionTotalMemoryLimit string, isPersistent bool, timeout time.Duration, cmdStarter internal.Starter) *TestSpace {
	testSpace := &TestSpace{
		QuotaDefinitionName:                  quotaDefinitionName,
		QuotaDefinitionTotalMemoryLimit:      quotaDefinitionTotalMemoryLimit,
		QuotaDefinitionInstanceMemoryLimit:   "-1",
		QuotaDefinitionRoutesLimit:           "1000",
		QuotaDefinitionAppInstanceLimit:      "-1",
		QuotaDefinitionServiceInstanceLimit:  "100",
		QuotaDefinitionAllowPaidServicesFlag: "--allow-paid-service-plans",
		organizationName:                     organizationName,
		spaceName:                            spaceName,
		CommandStarter:                       cmdStarter,
		Timeout:                              timeout,
		isPersistent:                         isPersistent,
	}
	return testSpace

}

func (ts *TestSpace) Create() {
	args := []string{
		"create-quota",
		ts.QuotaDefinitionName,
		"-m", ts.QuotaDefinitionTotalMemoryLimit,
		"-i", ts.QuotaDefinitionInstanceMemoryLimit,
		"-r", ts.QuotaDefinitionRoutesLimit,
		"-a", ts.QuotaDefinitionAppInstanceLimit,
		"-s", ts.QuotaDefinitionServiceInstanceLimit,
		ts.QuotaDefinitionAllowPaidServicesFlag,
	}

	createQuota := internal.Cf(ts.CommandStarter, args...)
	EventuallyWithOffset(1, createQuota, ts.Timeout).Should(Exit(0))

	createOrg := internal.Cf(ts.CommandStarter, "create-org", ts.organizationName)
	EventuallyWithOffset(1, createOrg, ts.Timeout).Should(Exit(0))

	setQuota := internal.Cf(ts.CommandStarter, "set-quota", ts.organizationName, ts.QuotaDefinitionName)
	EventuallyWithOffset(1, setQuota, ts.Timeout).Should(Exit(0))

	createSpace := internal.Cf(ts.CommandStarter, "create-space", "-o", ts.organizationName, ts.spaceName)
	EventuallyWithOffset(1, createSpace, ts.Timeout).Should(Exit(0))
}

func (ts *TestSpace) Destroy() {
	deleteOrg := internal.Cf(ts.CommandStarter, "delete-org", "-f", ts.organizationName)
	EventuallyWithOffset(1, deleteOrg, ts.Timeout).Should(Exit(0))

	deleteQuota := internal.Cf(ts.CommandStarter, "delete-quota", "-f", ts.QuotaDefinitionName)
	EventuallyWithOffset(1, deleteQuota, ts.Timeout).Should(Exit(0))
}

func (ts *TestSpace) OrganizationName() string {
	if ts == nil {
		return ""
	}
	return ts.organizationName
}

func (ts *TestSpace) SpaceName() string {
	if ts == nil {
		return ""
	}
	return ts.spaceName
}

func (ts *TestSpace) ShouldRemain() bool {
	return ts.isPersistent
}
