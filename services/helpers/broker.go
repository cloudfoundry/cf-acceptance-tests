package helpers

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

const brokerStartTimeout = 5 * 60.0
const defaultTimeout = 30

type ServiceBroker struct {
	Name    string
	Path    string
	context SuiteContext
	Service struct {
		Name            string `json:"name"`
		ID              string `json:"id"`
		DashboardClient struct {
			ID          string `json:"id"`
			Secret      string `json:"secret"`
			RedirectUri string `json:"redirect_uri"`
		}
	}
	Plan struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
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

func NewServiceBroker(name string, path string, context SuiteContext) ServiceBroker {
	b := ServiceBroker{}
	b.Path = path
	b.Name = name
	b.Service.Name = generator.RandomName()
	b.Service.ID = generator.RandomName()
	b.Plan.Name = generator.RandomName()
	b.Plan.ID = generator.RandomName()
	b.Service.DashboardClient.ID = generator.RandomName()
	b.Service.DashboardClient.Secret = generator.RandomName()
	b.Service.DashboardClient.RedirectUri = generator.RandomName()
	b.context = context
	return b
}

func (b ServiceBroker) Push() {
	Eventually(Cf("push", b.Name, "-p", b.Path), brokerStartTimeout).Should(Exit(0))
}

func (b ServiceBroker) Configure() {
	Eventually(Cf("set-env", b.Name, "CONFIG", b.ToJSON()), defaultTimeout).Should(Exit(0))
	b.Restart()
}

func (b ServiceBroker) Restart() {
	Eventually(Cf("restart", b.Name), brokerStartTimeout).Should(Exit(0))
}

func (b ServiceBroker) Create(appsDomain string) {
	AsUser(b.context.AdminUserContext(), func() {
		Eventually(Cf("create-service-broker", b.Name, "username", "password", AppUri(b.Name, "", appsDomain)), defaultTimeout).Should(Exit(0))
		Eventually(Cf("service-brokers"), defaultTimeout).Should(Say(b.Name))
	})
}

func (b ServiceBroker) Update(appsDomain string) {
	AsUser(b.context.AdminUserContext(), func() {
		Eventually(Cf("update-service-broker", b.Name, "username", "password", AppUri(b.Name, "", appsDomain)), defaultTimeout).Should(Exit(0))
	})
}

func (b ServiceBroker) Delete() {
	AsUser(b.context.AdminUserContext(), func() {
		Eventually(Cf("delete-service-broker", b.Name, "-f"), defaultTimeout).Should(Exit(0))
		Expect(Cf("service-brokers").Wait(defaultTimeout).Out.Contents()).ToNot(ContainSubstring(b.Name))
	})
}

func (b ServiceBroker) Destroy() {
	AsUser(b.context.AdminUserContext(), func() {
		Eventually(Cf("purge-service-offering", b.Service.Name, "-f"), defaultTimeout).Should(Exit(0))
	})
	b.Delete()
	Eventually(Cf("delete", b.Name, "-f"), defaultTimeout).Should(Exit(0))
}

func (b ServiceBroker) ToJSON() string {
	attributes := make(map[string]interface{})
	attributes["service"] = b.Service
	attributes["plan"] = b.Plan
	attributes["dashboard_client"] = b.Service.DashboardClient
	jsonBytes, _ := json.Marshal(attributes)
	return string(jsonBytes)
}

func (b ServiceBroker) PublicizePlans() {
	url := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=label:%s", b.Service.Name)
	var session *Session
	AsUser(b.context.AdminUserContext(), func() {
		session = Cf("curl", url).Wait(defaultTimeout)
	})
	structure := ServicesResponse{}
	json.Unmarshal(session.Out.Contents(), &structure)

	for _, service := range structure.Resources {
		if service.Entity.Label == b.Service.Name {
			for _, plan := range service.Entity.ServicePlans {
				if plan.Entity.Name == b.Plan.Name {
					b.PublicizePlan(plan.Metadata.Url)
					break
				}
			}
		}
	}
}

func (b ServiceBroker) PublicizePlan(url string) {
	jsonMap := make(map[string]bool)
	jsonMap["public"] = true
	planJson, _ := json.Marshal(jsonMap)
	AsUser(b.context.AdminUserContext(), func() {
		Eventually(Cf("curl", url, "-X", "PUT", "-d", string(planJson)), defaultTimeout).Should(Exit(0))
	})
}

func (b ServiceBroker) CreateServiceInstance(instanceName string) (guid string) {
  Eventually(Cf("create-service", b.Service.Name, b.Plan.Name, instanceName), defaultTimeout).Should(Exit(0))
	url := fmt.Sprintf("/v2/service_instances?q=name:%s", instanceName)
	apiResponse := Cf("curl", url).Wait(defaultTimeout).Out.Contents()

  serviceInstance := ServiceInstanceResponse{}
	json.Unmarshal(apiResponse, &serviceInstance)

	guid = serviceInstance.Resources[0].Metadata.Guid
	return
}

func (b ServiceBroker) GetSpaceGuid() string {
	url := fmt.Sprintf("/v2/spaces?q=name%%3A%s", b.context.RegularUserContext().Space)
	jsonResults := SpaceJson{}
	json.Unmarshal(Cf("curl", url).Wait(defaultTimeout).Out.Contents(), &jsonResults)
	return jsonResults.Resources[0].Metadata.Guid
}
