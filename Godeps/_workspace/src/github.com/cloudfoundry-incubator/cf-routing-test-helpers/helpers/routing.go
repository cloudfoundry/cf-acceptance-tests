package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-routing-test-helpers/schema"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

const (
	DEFAULT_MEMORY_LIMIT = "256M"
	deaUnsupportedTag    = "{NO_DEA_SUPPORT} "
)

func MapRouteToApp(app, domain, host, path string, timeout time.Duration) {
	Expect(cf.Cf("map-route", app, domain, "--hostname", host, "--path", path).Wait(timeout)).To(Exit(0))
}

func DeleteRoute(hostname, contextPath, domain string, timeout time.Duration) {
	Expect(cf.Cf("delete-route", domain,
		"--hostname", hostname,
		"--path", contextPath,
		"-f",
	).Wait(timeout)).To(Exit(0))
}

func CreateRoute(hostname, contextPath, space, domain string, timeout time.Duration) {
	Expect(cf.Cf("create-route", space, domain,
		"--hostname", hostname,
		"--path", contextPath,
	).Wait(timeout)).To(Exit(0))
}

func GetRouteGuid(hostname, path string, timeout time.Duration) string {
	responseBuffer := cf.Cf("curl", fmt.Sprintf("/v2/routes?q=host:%s&q=path:%s", hostname, path))
	Expect(responseBuffer.Wait(timeout)).To(Exit(0))
	routeBytes := responseBuffer.Out.Contents()

	var routeResponse schema.ListResponse

	err := json.Unmarshal(routeBytes, &routeResponse)
	Expect(err).NotTo(HaveOccurred())
	Expect(routeResponse.TotalResults).To(Equal(1))

	return routeResponse.Resources[0].Metadata.Guid
}

func GetAppInfo(appName string, timeout time.Duration) (host, port string) {
	var appsResponse schema.AppsResponse
	var statsResponse schema.StatsResponse

	cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait(timeout).Out.Contents()
	err := json.Unmarshal(cfResponse, &appsResponse)
	Expect(err).NotTo(HaveOccurred())
	serverAppUrl := appsResponse.Resources[0].Metadata.Url

	cfResponse = cf.Cf("curl", fmt.Sprintf("%s/stats", serverAppUrl)).Wait(timeout).Out.Contents()
	err = json.Unmarshal(cfResponse, &statsResponse)
	Expect(err).NotTo(HaveOccurred())

	appIp := statsResponse["0"].Stats.Host
	appPort := fmt.Sprintf("%d", statsResponse["0"].Stats.Port)
	return appIp, appPort
}

func UpdatePorts(appName string, ports []uint32, timeout time.Duration) {
	appGuid := GetAppGuid(appName, timeout)

	bodyMap := map[string][]uint32{
		"ports": ports,
	}

	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())

	Expect(cf.Cf("curl", fmt.Sprintf("/v2/apps/%s", appGuid), "-X", "PUT", "-d", string(data)).Wait(timeout)).To(Exit(0))
}

func CreateRouteMapping(appName string, hostname string, port uint32, timeout time.Duration) {
	appGuid := GetAppGuid(appName, timeout)
	routeGuid := GetRouteGuid(hostname, "", timeout)

	bodyMap := map[string]interface{}{
		"app_guid":   appGuid,
		"route_guid": routeGuid,
		"app_port":   port,
	}

	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())

	Expect(cf.Cf("curl", fmt.Sprintf("/v2/route_mappings"), "-X", "POST", "-d", string(data)).Wait(timeout)).To(Exit(0))
}
