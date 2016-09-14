package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/cf-routing-test-helpers/schema"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

const (
	deaUnsupportedTag = "{NO_DEA_SUPPORT} "
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

func CreateTcpRouteWithRandomPort(space, domain string, timeout time.Duration) uint16 {
	var routeResponse schema.RouteResource

	domainGuid := GetDomainGuid(domain, timeout)
	spaceGuid := GetSpaceGuid(space, timeout)

	bodyMap := map[string]interface{}{
		"domain_guid": domainGuid,
		"space_guid":  spaceGuid,
	}
	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())

	responseBuffer := cf.Cf("curl", "/v2/routes?generate_port=true", "-X", "POST", "-d", string(data))
	Expect(responseBuffer.Wait(timeout)).To(Exit(0))

	err = json.Unmarshal(responseBuffer.Out.Contents(), &routeResponse)
	Expect(err).NotTo(HaveOccurred())
	Expect(routeResponse.Entity.Port).NotTo(BeZero())
	return routeResponse.Entity.Port
}

func GetGuid(curlPath string, timeout time.Duration) string {
	var response schema.ListResponse

	responseBuffer := cf.Cf("curl", curlPath)
	Expect(responseBuffer.Wait(timeout)).To(Exit(0))

	err := json.Unmarshal(responseBuffer.Out.Contents(), &response)
	Expect(err).NotTo(HaveOccurred())
	if response.TotalResults == 1 {
		return response.Resources[0].Metadata.Guid
	}
	return ""
}

func GetDomainGuid(domainName string, timeout time.Duration) string {
	sharedDomainGuid := GetGuid(fmt.Sprintf("/v2/shared_domains?q=name:%s", domainName), timeout)
	if sharedDomainGuid != "" {
		return sharedDomainGuid
	}

	privateDomainGuid := GetGuid(fmt.Sprintf("/v2/private_domains?q=name:%s", domainName), timeout)
	Expect(privateDomainGuid).ToNot(Equal(""))
	return privateDomainGuid
}

func GetSpaceGuid(space string, timeout time.Duration) string {
	spaceGuid := GetGuid(fmt.Sprintf("/v2/spaces?q=name:%s", space), timeout)
	Expect(spaceGuid).NotTo(Equal(""))
	return spaceGuid
}

func GetRouteGuidWithPort(hostname, path string, port uint16, timeout time.Duration) string {
	routeQuery := fmt.Sprintf("/v2/routes?q=host:%s&q=path:%s", hostname, path)
	if port > 0 {
		routeQuery = routeQuery + fmt.Sprintf("&q=port:%d", port)
	}
	routeGuid := GetGuid(routeQuery, timeout)
	Expect(routeGuid).NotTo(Equal(""))
	return routeGuid
}

func GetRouteGuid(hostname, path string, timeout time.Duration) string {
	return GetRouteGuidWithPort(hostname, path, 0, timeout)
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

func UpdatePorts(appName string, ports []uint16, timeout time.Duration) {
	appGuid := GetAppGuid(appName, timeout)

	bodyMap := map[string][]uint16{
		"ports": ports,
	}

	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())

	Expect(cf.Cf("curl", fmt.Sprintf("/v2/apps/%s", appGuid), "-X", "PUT", "-d", string(data)).Wait(timeout)).To(Exit(0))
}

func CreateRouteMapping(appName string, hostname string, externalPort uint16, appPort uint16, timeout time.Duration) {
	appGuid := GetAppGuid(appName, timeout)
	routeGuid := GetRouteGuidWithPort(hostname, "", externalPort, timeout)

	bodyMap := map[string]interface{}{
		"app_guid":   appGuid,
		"route_guid": routeGuid,
		"app_port":   appPort,
	}

	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())

	Expect(cf.Cf("curl", fmt.Sprintf("/v2/route_mappings"), "-X", "POST", "-d", string(data)).Wait(timeout)).To(Exit(0))
}

func CreateSharedDomain(domainName, routerGroupGuid string, timeout time.Duration) {
	bodyMap := map[string]interface{}{
		"name":              domainName,
		"router_group_guid": routerGroupGuid,
	}

	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())
	resp := cf.Cf("curl", fmt.Sprintf("/v2/shared_domains"), "-X", "POST", "-d", string(data))
	resp.Wait(timeout)
}

func DeleteSharedDomain(domainName string, timeout time.Duration) {
	sharedDomainGuid := GetGuid(fmt.Sprintf("/v2/shared_domains?q=name:%s", domainName), timeout)
	Expect(cf.Cf("curl", "-X", "DELETE", "/v2/shared_domains/"+sharedDomainGuid).Wait(timeout)).To(Exit(0))
}
