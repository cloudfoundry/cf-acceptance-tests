package volume_services

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = VolumeServicesDescribe("Volume Services", func() {

	var (
		serviceName         string
		serviceInstanceName string
		appName             string
		poraAsset           = assets.NewAssets().Pora
		routerGroupGuid     string
		reservablePorts     string
		nfsPort             = "2049"
	)

	BeforeEach(func() {
		serviceName = Config.GetVolumeServiceName()
		serviceInstanceName = random_name.CATSRandomName("SVIN")
		appName = random_name.CATSRandomName("APP")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("curl", "/routing/v1/router_groups").Wait()
			Expect(session).To(Exit(0), "cannot retrieve current router groups")

			routerGroupGuid, reservablePorts = routerGroupIdAndPorts(session.Out.Contents())

			payload := `{ "reservable_ports":"1024-2049", "name":"default-tcp", "type": "tcp"}`
			session = cf.Cf("curl", fmt.Sprintf("/routing/v1/router_groups/%s", routerGroupGuid), "-X", "PUT", "-d", payload).Wait()
			Expect(session).To(Exit(0), "cannot update tcp router group to allow nfs traffic")
		})

		By("pushing an nfs server")
		Expect(cf.Cf("push", "nfs", "--docker-image", "cfpersi/nfs-cats", "--health-check-type", "process", "--no-start").
			Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "cannot push the nfs server app")

		tcpDomain := fmt.Sprintf("tcp.%s", Config.GetAppsDomain())
		session := cf.Cf("create-route", TestSetup.RegularUserContext().Space, tcpDomain, "--port", nfsPort).Wait()
		Expect(session).To(Exit(0), "cannot create a tcp route for the nfs server app")

		nfsGuid := GuidForAppName("nfs")
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("curl", "/v2/routes").Wait()
			Expect(session).To(Exit(0), "cannot retrieve current routes")

			routes := &Routes{}
			err := json.Unmarshal(session.Out.Contents(), routes)
			Expect(err).NotTo(HaveOccurred())

			routeId := nfsRouteGuid(routes)

			session = cf.Cf("curl", "/v2/route_mappings", "-X", "POST", "-d", fmt.Sprintf(`{"app_guid": "%s", "route_guid": "%s", "app_port": %s}`, nfsGuid, routeId, nfsPort)).Wait()
			Expect(session).To(Exit(0), "cannot create a tcp route mapping to the nfs server app")
		})

		session = cf.Cf("start", "nfs").Wait()
		Expect(session).To(Exit(0), "cannot start the nfs server app")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("enable-service-access", serviceName, "-o", TestSetup.RegularUserContext().Org).Wait()
			Expect(session).To(Exit(0), "cannot enable nfs service access")
		})

		By("pushing an app")
		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", poraAsset,
			"-f", filepath.Join(poraAsset, "manifest.yml"),
			"-d", Config.GetAppsDomain(),
			"--no-start",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "cannot push the test app")

		By("creating a service")
		var createServiceSession *Session
		if Config.GetVolumeServiceCreateConfig() != "" {
			createServiceSession = cf.Cf("create-service", serviceName, Config.GetVolumeServicePlanName(), serviceInstanceName, "-c", fmt.Sprintf(`%s`, Config.GetVolumeServiceCreateConfig()))
		} else {
			createServiceSession = cf.Cf("create-service", serviceName, Config.GetVolumeServicePlanName(), serviceInstanceName, "-c", fmt.Sprintf(`{"share": "%s/"}`, tcpDomain))
		}
		Expect(createServiceSession.Wait(TestSetup.ShortTimeout())).To(Exit(0), "cannot create an nfs service instance")

		By("binding the service")
		var bindSession *Session
		if Config.GetVolumeServiceCreateConfig() != "" {
			bindSession = cf.Cf("bind-service", appName, serviceInstanceName, "cannot bind service to app")
		} else {
			bindSession = cf.Cf("bind-service", appName, serviceInstanceName, "-c", `{"uid": "2000", "gid": "2000"}`, "cannot bind nfs service to app")
		}
		Expect(bindSession.Wait(TestSetup.ShortTimeout())).To(Exit(0), "cannot bind the nfs service instance to the test app")

		By("starting the app")
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "cannot start the test app")
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0), "cannot delete the test app")
		Eventually(cf.Cf("delete-service", serviceInstanceName, "-f")).Should(Exit(0), "cannot delete the nfs service instance")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("disable-service-access", serviceName, "-o", TestSetup.RegularUserContext().Org).Wait()
			Expect(session).To(Exit(0), "cannot disable nfs service access")
		})

		Eventually(cf.Cf("delete", "nfs", "-f")).Should(Exit(0), "cannot delete the nfs server app")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			payload := fmt.Sprintf(`{ "reservable_ports":"%s", "name":"default-tcp", "type": "tcp"}`, reservablePorts)
			session := cf.Cf("curl", fmt.Sprintf("/routing/v1/router_groups/%s", routerGroupGuid), "-X", "PUT", "-d", payload).Wait()
			Expect(session).To(Exit(0), "cannot retrieve current router groups")
		})
	})

	It("should be able to write to the volume", func() {
		Expect(helpers.CurlApp(Config, appName, "/write")).To(ContainSubstring("Hello Persistent World"))
	})
})

func nfsRouteGuid(routes *Routes) string {
	for _, resource := range routes.Resources {
		if resource.Entity.Port != nil && resource.Entity.Port.(float64) == 2049 {
			return resource.Metadata.GUID
		}
	}
	Fail("Unable to find a valid tcp route for port 2049")
	return ""
}

func routerGroupIdAndPorts(routerGroupOutput []byte) (guid, ports string) {
	routerGroups := &[]RouterGroup{}
	err := json.Unmarshal(routerGroupOutput, routerGroups)
	Expect(err).NotTo(HaveOccurred())
	for _, routerGroup := range *routerGroups {
		if routerGroup.Name == "default-tcp" {
			return routerGroup.GUID, routerGroup.ReservablePorts
		}
	}
	Fail("Unable to find routergroup 'default-tcp'")
	return "", ""
}

type RouterGroup struct {
	GUID            string `json:"guid"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	ReservablePorts string `json:"reservable_ports"`
}

type Routes struct {
	TotalResults int         `json:"total_results"`
	TotalPages   int         `json:"total_pages"`
	PrevURL      interface{} `json:"prev_url"`
	NextURL      interface{} `json:"next_url"`
	Resources    []struct {
		Metadata struct {
			GUID      string    `json:"guid"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"metadata"`
		Entity struct {
			Host                string      `json:"host"`
			Path                string      `json:"path"`
			DomainGUID          string      `json:"domain_guid"`
			SpaceGUID           string      `json:"space_guid"`
			ServiceInstanceGUID interface{} `json:"service_instance_guid"`
			Port                interface{} `json:"port"`
			DomainURL           string      `json:"domain_url"`
			SpaceURL            string      `json:"space_url"`
			AppsURL             string      `json:"apps_url"`
			RouteMappingsURL    string      `json:"route_mappings_url"`
		} `json:"entity"`
	} `json:"resources"`
}
