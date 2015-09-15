package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

type Plan struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type ServiceBroker struct {
	Name    string
	Path    string
	context helpers.SuiteContext
	Service struct {
		Name            string `json:"name"`
		ID              string `json:"id"`
		DashboardClient struct {
			ID          string `json:"id"`
			Secret      string `json:"secret"`
			RedirectUri string `json:"redirect_uri"`
		}
	}
	SyncPlans  []Plan
	AsyncPlans []Plan
}

type ServicesResponse struct {
	Resources []ServiceResponse
}

type ServiceResponse struct {
	Entity struct {
		Label        string
		ServicePlans []ServicePlanResponse `json:"service_plans"`
	}
}

type ServicePlanResponse struct {
	Entity struct {
		Name   string
		Public bool
	}
	Metadata struct {
		Url  string
		Guid string
	}
}

type ServiceInstance struct {
	Metadata struct {
		Guid string `json:"guid"`
	}
}

type ServiceInstanceResponse struct {
	Resources []ServiceInstance
}

type SpaceJson struct {
	Resources []struct {
		Metadata struct {
			Guid string
		}
	}
}

func NewServiceBroker(name string, path string, context helpers.SuiteContext) ServiceBroker {
	b := ServiceBroker{}
	b.Path = path
	b.Name = name
	b.Service.Name = generator.RandomName()
	b.Service.ID = generator.RandomName()
	b.SyncPlans = []Plan{
		{Name: generator.RandomName(), ID: generator.RandomName()},
		{Name: generator.RandomName(), ID: generator.RandomName()},
	}
	b.AsyncPlans = []Plan{
		{Name: generator.RandomName(), ID: generator.RandomName()},
		{Name: generator.RandomName(), ID: generator.RandomName()},
	}
	b.Service.DashboardClient.ID = generator.RandomName()
	b.Service.DashboardClient.Secret = generator.RandomName()
	b.Service.DashboardClient.RedirectUri = generator.RandomName()
	b.context = context
	return b
}

func (b ServiceBroker) Push() {
	Expect(cf.Cf("push", b.Name, "-m", "128M", "-p", b.Path, "--no-start").Wait(BROKER_START_TIMEOUT)).To(Exit(0))
	if helpers.LoadConfig().UseDiego {
		appGuid := strings.TrimSpace(string(cf.Cf("app", b.Name, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()))
		cf.Cf("curl",
			fmt.Sprintf("/v2/apps/%s", appGuid),
			"-X", "PUT",
			"-d", "{\"diego\": true}",
		).Wait(DEFAULT_TIMEOUT)
	}
	Expect(cf.Cf("start", b.Name).Wait(BROKER_START_TIMEOUT)).To(Exit(0))
}

func (b ServiceBroker) Configure() {
	Expect(runner.Curl(helpers.AppUri(b.Name, "/config"), "-d", b.ToJSON()).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func (b ServiceBroker) Restart() {
	Expect(cf.Cf("restart", b.Name).Wait(BROKER_START_TIMEOUT)).To(Exit(0))
}

func (b ServiceBroker) Create() {
	cf.AsUser(b.context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		Expect(cf.Cf("create-service-broker", b.Name, "username", "password", helpers.AppUri(b.Name, "")).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)).To(Say(b.Name))
	})
}

func (b ServiceBroker) Update() {
	cf.AsUser(b.context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		Expect(cf.Cf("update-service-broker", b.Name, "username", "password", helpers.AppUri(b.Name, "")).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func (b ServiceBroker) Delete() {
	cf.AsUser(b.context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		Expect(cf.Cf("delete-service-broker", b.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		brokers := cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)
		Expect(brokers).To(Exit(0))
		Expect(brokers.Out.Contents()).ToNot(ContainSubstring(b.Name))
	})
}

func (b ServiceBroker) Destroy() {
	cf.AsUser(b.context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		Expect(cf.Cf("purge-service-offering", b.Service.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
	b.Delete()
	Expect(cf.Cf("delete", b.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func (b ServiceBroker) ToJSON() string {
	bytes, err := ioutil.ReadFile(assets.NewAssets().ServiceBroker + "/cats.json")
	Expect(err).To(BeNil())

	replacer := strings.NewReplacer(
		"<fake-service>", b.Service.Name,
		"<fake-service-guid>", b.Service.ID,
		"<sso-test>", b.Service.DashboardClient.ID,
		"<sso-secret>", b.Service.DashboardClient.Secret,
		"<sso-redirect-uri>", b.Service.DashboardClient.RedirectUri,
		"<fake-plan>", b.SyncPlans[0].Name,
		"<fake-plan-guid>", b.SyncPlans[0].ID,
		"<fake-plan-2>", b.SyncPlans[1].Name,
		"<fake-plan-2-guid>", b.SyncPlans[1].ID,
		"<fake-async-plan>", b.AsyncPlans[0].Name,
		"<fake-async-plan-guid>", b.AsyncPlans[0].ID,
		"<fake-async-plan-2>", b.AsyncPlans[1].Name,
		"<fake-async-plan-2-guid>", b.AsyncPlans[1].ID,
	)

	return replacer.Replace(string(bytes))
}

func (b ServiceBroker) PublicizePlans() {
	url := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=label:%s", b.Service.Name)
	var session *Session
	cf.AsUser(b.context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		session = cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
		Expect(session).To(Exit(0))
	})
	structure := ServicesResponse{}
	json.Unmarshal(session.Out.Contents(), &structure)

	for _, service := range structure.Resources {
		if service.Entity.Label == b.Service.Name {
			for _, plan := range service.Entity.ServicePlans {
				if b.HasPlan(plan.Entity.Name) {
					b.PublicizePlan(plan.Metadata.Url)
				}
			}
		}
	}
}

func (b ServiceBroker) HasPlan(planName string) bool {
	for _, plan := range b.Plans() {
		if plan.Name == planName {
			return true
		}
	}
	return false
}

func (b ServiceBroker) PublicizePlan(url string) {
	jsonMap := make(map[string]bool)
	jsonMap["public"] = true
	planJson, _ := json.Marshal(jsonMap)
	cf.AsUser(b.context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		Expect(cf.Cf("curl", url, "-X", "PUT", "-d", string(planJson)).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func (b ServiceBroker) CreateServiceInstance(instanceName string) string {
	Expect(cf.Cf("create-service", b.Service.Name, b.SyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	url := fmt.Sprintf("/v2/service_instances?q=name:%s", instanceName)
	serviceInstance := ServiceInstanceResponse{}
	curl := cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
	Expect(curl).To(Exit(0))
	json.Unmarshal(curl.Out.Contents(), &serviceInstance)
	return serviceInstance.Resources[0].Metadata.Guid
}

func (b ServiceBroker) GetSpaceGuid() string {
	url := fmt.Sprintf("/v2/spaces?q=name%%3A%s", b.context.RegularUserContext().Space)
	jsonResults := SpaceJson{}
	curl := cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
	Expect(curl).To(Exit(0))
	json.Unmarshal(curl.Out.Contents(), &jsonResults)
	return jsonResults.Resources[0].Metadata.Guid
}

func (b ServiceBroker) Plans() []Plan {
	plans := make([]Plan, 0)
	plans = append(plans, b.SyncPlans...)
	plans = append(plans, b.AsyncPlans...)
	return plans
}
