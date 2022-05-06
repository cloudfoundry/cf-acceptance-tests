package v3

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/cf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = V3Describe("service instances", func() {
	var (
		broker               services.ServiceBroker
		serviceInstance1Name string
		serviceInstance2Name string
	)

	type ServiceInstance struct {
		Name string
	}
	type Response struct {
		Resources []ServiceInstance
	}

	BeforeEach(func() {
		broker = services.NewServiceBroker(
			random_name.CATSRandomName("BRKR"),
			assets.NewAssets().ServiceBroker,
			TestSetup,
		)
		broker.Push(Config)
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		serviceInstance1Name = random_name.CATSRandomName("SVIN")
		serviceInstance2Name = random_name.CATSRandomName("SVIN")

		By("Creating a service instance")
		createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstance1Name).Wait()
		Expect(createService).To(Exit(0))

		By("Creating another service instance")
		createService2 := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstance2Name).Wait()
		Expect(createService2).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(broker.Name)

		Expect(cf.Cf("delete-service", serviceInstance1Name, "-f").Wait()).To(Exit(0))
		Expect(cf.Cf("delete-service", serviceInstance2Name, "-f").Wait()).To(Exit(0))

		broker.Destroy()
	})

	It("Lists the service instances", func() {
		expectedResources := []ServiceInstance{
			{Name: serviceInstance1Name},
			{Name: serviceInstance2Name},
		}

		spaceGuid := GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		listService := cf.Cf("curl", fmt.Sprintf("/v3/service_instances?space_guids=%s", spaceGuid)).Wait()
		Expect(listService).To(Exit(0))

		var res Response
		err := json.Unmarshal(listService.Out.Contents(), &res)
		Expect(err).To(BeNil())

		Expect(res.Resources).To(ConsistOf(expectedResources))
	})
})
