package volume_services

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-test-helpers/workflowhelpers"
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
		tcpDomain           string
	)

	BeforeEach(func() {
		serviceName = Config.GetVolumeServiceName()
		serviceInstanceName = random_name.CATSRandomName("SVIN")
		appName = random_name.CATSRandomName("APP")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("enable-feature-flag", "diego_docker").Wait()
			Expect(session).To(Exit(0), "cannot enable diego_docker feature flag")

			session = cf.Cf("curl", "/routing/v1/router_groups").Wait()
			Expect(session).To(Exit(0), "cannot retrieve current router groups")

			routerGroupGuid, reservablePorts = routerGroupIdAndPorts(session.Out.Contents())

			payload := `{ "reservable_ports":"1024-2049", "name":"default-tcp", "type": "tcp"}`
			session = cf.Cf("curl", fmt.Sprintf("/routing/v1/router_groups/%s", routerGroupGuid), "-X", "PUT", "-d", payload).Wait()
			Expect(session).To(Exit(0), "cannot update tcp router group to allow nfs traffic")

			tcpDomain = fmt.Sprintf("tcp.%s", Config.GetAppsDomain())

			session = cf.Cf("create-shared-domain", tcpDomain, "--router-group", "default-tcp").Wait()
			Eventually(session).Should(Exit())
			contents := string(session.Out.Contents()) + string(session.Err.Contents())
			Expect(contents).Should(
				SatisfyAny(
					ContainSubstring(fmt.Sprintf("The domain name %q is already in use", tcpDomain)),
					ContainSubstring("OK"),
				), "can not create shared tcp domain >>>"+contents)

		})

		By("pushing an nfs server")
		Expect(cf.Cf("push", "nfs", "--docker-image", "cfpersi/nfs-cats", "--health-check-type", "process", "--no-start").
			Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "cannot push the nfs server app")

		session := cf.Cf("create-route", tcpDomain, "--port", nfsPort).Wait()
		Expect(session).To(Exit(0), "cannot create a tcp route for the nfs server app")

		nfsGuid := GuidForAppName("nfs")
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("curl", "/v3/routes").Wait()
			Expect(session).To(Exit(0), "cannot retrieve current routes")

			routes := &Routes{}
			err := json.Unmarshal(session.Out.Contents(), routes)
			Expect(err).NotTo(HaveOccurred())

			routeId := nfsRouteGuid(routes)

			session = cf.Cf("curl", fmt.Sprintf("/v3/routes/%s/destinations", routeId), "-X", "POST", "-d", fmt.Sprintf(`{"destinations": [{"app": {"guid": "%s"}, "port": %s}]}`, nfsGuid, nfsPort)).Wait()
			Expect(session).To(Exit(0), "cannot create a tcp route mapping to the nfs server app")
		})

		session = cf.Cf("start", "nfs").Wait(Config.CfPushTimeoutDuration())
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
			"--no-start",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "cannot push the test app")

		By("creating a service")
		var createServiceSession *Session
		if Config.GetVolumeServiceCreateConfig() != "" {
			createServiceSession = cf.Cf("create-service", serviceName, Config.GetVolumeServicePlanName(), serviceInstanceName, "-c", Config.GetVolumeServiceCreateConfig())
		} else {
			createServiceSession = cf.Cf("create-service", serviceName, Config.GetVolumeServicePlanName(), serviceInstanceName, "-c", fmt.Sprintf(`{"share": "%s/"}`, tcpDomain))
		}
		Expect(createServiceSession.Wait(TestSetup.ShortTimeout())).To(Exit(0), "cannot create an nfs service instance")

		By("binding the service")
		var bindSession *Session
		if Config.GetVolumeServiceCreateConfig() != "" {
			bindSession = cf.Cf("bind-service", appName, serviceInstanceName)
		} else {
			bindSession = cf.Cf("bind-service", appName, serviceInstanceName, "-c", `{"uid": "2000", "gid": "2000"}`)
		}
		Expect(bindSession.Wait(TestSetup.ShortTimeout())).To(Exit(0), "cannot bind the nfs service instance to the test app")

		By("starting the app")
		session = cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
		Eventually(session).Should(Exit())
		if session.ExitCode() != 0 {
			cf.Cf("logs", appName, "--recent")
		}
		Expect(session.ExitCode()).To(Equal(0))
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			payload := fmt.Sprintf(`{ "reservable_ports":"%s", "name":"default-tcp", "type": "tcp"}`, reservablePorts)
			session := cf.Cf("curl", fmt.Sprintf("/routing/v1/router_groups/%s", routerGroupGuid), "-X", "PUT", "-d", payload).Wait()
			Expect(session).To(Exit(0), "cannot retrieve current router groups")

			session = cf.Cf("disable-feature-flag", "diego_docker").Wait()
			Expect(session).To(Exit(0), "cannot disable diego_docker feature flag")
		})
	})

	It("should be able to write to the volume", func() {
		Expect(helpers.CurlApp(Config, appName, "/write")).To(ContainSubstring("Hello Persistent World"))
	})
})

func nfsRouteGuid(routes *Routes) string {
	for _, resource := range routes.Resources {
		if resource.Port != 0 && resource.Port == 2049 {
			return resource.GUID
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
	TotalResults int `json:"total_results"`
	TotalPages   int `json:"total_pages"`
	Previous     struct {
		Href string `json:"prev_url"`
	}
	Next struct {
		Href string `json:"prev_url"`
	}
	Resources []struct {
		GUID         string    `json:"guid"`
		URL          string    `json:"url"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Host         string    `json:"host"`
		Path         string    `json:"path"`
		Port         int       `json:"port"`
		Destinations []struct {
			GUID string `json:"guid"`
			App  struct {
				GUID    string `json:"guid"`
				Port    int    `json:"port"`
				Process struct {
					Type string `json:"type"`
				} `json:"process"`
			} `json:"app"`
			ServiceInstance struct {
				GUID string `json:"guid"`
			} `json:"service_instance"`
		} `json:"destinations"`
		Relationships struct {
			Space struct {
				Data struct {
					GUID string `json:"guid"`
				} `json:"data"`
			} `json:"space"`
			Domain struct {
				Data struct {
					GUID string `json:"guid"`
				} `json:"data"`
			} `json:"domain"`
		} `json:"relationships"`
	} `json:"resources"`
}
