package volume_services

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
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
		tcpDomain           string
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

			tcpDomain = Config.GetTCPDomain()

			session = cf.Cf("create-shared-domain", tcpDomain, "--router-group", "default-tcp").Wait()
			Eventually(session).Should(Exit())
			contents := string(session.Out.Contents()) + string(session.Err.Contents())
			Expect(contents).Should(
				SatisfyAny(
					ContainSubstring(fmt.Sprintf("The domain name %q is already in use", tcpDomain)),
					ContainSubstring("OK"),
				), "can not create shared tcp domain >>>"+contents)

		})

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			args := []string{"enable-service-access", serviceName, "-o", TestSetup.RegularUserContext().Org}
			if Config.GetVolumeServiceBrokerName() != "" {
				args = append(args, "-b", Config.GetVolumeServiceBrokerName())
			}
			session := cf.Cf(args...).Wait()
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
		args := []string{"create-service", serviceName, Config.GetVolumeServicePlanName(), serviceInstanceName}
		if Config.GetVolumeServiceBrokerName() != "" {
			args = append(args, "-b", Config.GetVolumeServiceBrokerName())
		}
		if Config.GetVolumeServiceCreateConfig() == "" {
			args = append(args, "-c", fmt.Sprintf(`{"share": "%s/"}`, tcpDomain))
		} else {
			args = append(args, "-c", Config.GetVolumeServiceCreateConfig())
		}
		createServiceSession = cf.Cf(args...)
		Expect(createServiceSession.Wait(TestSetup.ShortTimeout())).To(Exit(0), "cannot create an nfs service instance")

		By("binding the service")
		var bindSession *Session
		if Config.GetVolumeServiceBindConfig() == "" {
			bindSession = cf.Cf("bind-service", appName, serviceInstanceName, "-c", `{"uid": "1000", "gid": "1000"}`)
		} else {
			bindSession = cf.Cf("bind-service", appName, serviceInstanceName, "-c", Config.GetVolumeServiceBindConfig())
		}
		Expect(bindSession.Wait(TestSetup.ShortTimeout())).To(Exit(0), "cannot bind the nfs service instance to the test app")

		By("starting the app")
		session := cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
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
		})
	})

	It("should be able to write to the volume", func() {
		Expect(helpers.CurlApp(Config, appName, "/write")).To(ContainSubstring("Hello Persistent World"))
	})
})

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
