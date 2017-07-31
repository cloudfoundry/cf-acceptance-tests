package v3_helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const (
	V3_DEFAULT_MEMORY_LIMIT = "256"
	V3_JAVA_MEMORY_LIMIT    = "512"
)

func AssignDropletToApp(appGuid, dropletGuid string) {
	appUpdatePath := fmt.Sprintf("/v3/apps/%s/relationships/current_droplet", appGuid)
	appUpdateBody := fmt.Sprintf(`{"data": {"guid":"%s"}}`, dropletGuid)
	Expect(cf.Cf("curl", appUpdatePath, "-X", "PATCH", "-d", appUpdateBody).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

	for _, process := range GetProcesses(appGuid, "") {
		ScaleProcess(appGuid, process.Type, V3_DEFAULT_MEMORY_LIMIT)
	}
}

func AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl", fmt.Sprintf("/v3/spaces/%s/relationships/isolation_segment", spaceGuid),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func CreateAndMapRoute(appGuid, space, domain, host string) {
	CreateRoute(space, domain, host)
	getRoutePath := fmt.Sprintf("/v2/routes?q=host:%s", host)
	routeBody := cf.Cf("curl", getRoutePath).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	routeJSON := struct {
		Resources []struct {
			Metadata struct {
				Guid string `json:"guid"`
			} `json:"metadata"`
		} `json:"resources"`
	}{}
	json.Unmarshal([]byte(routeBody), &routeJSON)
	routeGuid := routeJSON.Resources[0].Metadata.Guid
	addRouteBody := fmt.Sprintf(`
	{
		"relationships": {
			"app":   {"guid": "%s"},
			"route": {"guid": "%s"}
		}
	}`, appGuid, routeGuid)
	Expect(cf.Cf("curl", "/v3/route_mappings", "-X", "POST", "-d", addRouteBody).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
}

func CreateApp(appName, spaceGuid, environmentVariables string) string {
	session := cf.Cf("curl", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables":%s}`, appName, spaceGuid, environmentVariables))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	var app struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &app)
	return app.Guid
}

func CreateDockerApp(appName, spaceGuid, environmentVariables string) string {
	session := cf.Cf("curl", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "relationships": {"space": {"data": {"guid": "%s"}}}, "environment_variables":%s, "lifecycle": {"type": "docker", "data": {} } }`, appName, spaceGuid, environmentVariables))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	var app struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &app)
	return app.Guid
}

func CreateDockerPackage(appGuid, imagePath string) string {
	packageCreateUrl := fmt.Sprintf("/v3/packages")
	session := cf.Cf("curl", packageCreateUrl, "-X", "POST", "-d", fmt.Sprintf(`{"relationships":{"app":{"data":{"guid":"%s"}}},"type":"docker", "data": {"image": "%s"}}`, appGuid, imagePath))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	var pac struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &pac)
	return pac.Guid
}

func CreateIsolationSegment(name string) string {
	session := cf.Cf("curl", "/v3/isolation_segments", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s"}`, name))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()

	var isolation_segment struct {
		Guid string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &isolation_segment)
	Expect(err).ToNot(HaveOccurred())

	return isolation_segment.Guid
}

func CreateOrGetIsolationSegment(name string) string {
	if IsolationSegmentExists(name) {
		return GetIsolationSegmentGuid(name)
	}

	return CreateIsolationSegment(name)
}

func CreatePackage(appGuid string) string {
	packageCreateUrl := fmt.Sprintf("/v3/packages")
	session := cf.Cf("curl", packageCreateUrl, "-X", "POST", "-d", fmt.Sprintf(`{"relationships":{"app":{"data":{"guid":"%s"}}},"type":"bits"}`, appGuid))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	var pac struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &pac)
	return pac.Guid
}

func CreateRoute(space, domain, host string) {
	Expect(cf.Cf("create-route", space, domain, "-n", host).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
}

func HandleAsyncRequest(path string, method string) {
	session := cf.Cf("curl", path, "-X", method, "-i")
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	Expect(string(bytes)).To(ContainSubstring("202 Accepted"))

	jobPath := getJobPath(bytes)
	pollJob(jobPath)
}

func getJobPath(response []byte) string {
	r, err := regexp.Compile(`Location:.*(/v3/jobs/[\w-]*)`)
	Expect(err).ToNot(HaveOccurred())
	return r.FindStringSubmatch(string(response))[1]
}

func pollJob(jobPath string) {
	Eventually(func() string {
		jobSession := cf.Cf("curl", jobPath)
		return string(jobSession.Wait(Config.DefaultTimeoutDuration()).Out.Contents())
	}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("COMPLETE"))
}

func DeleteApp(appGuid string) {
	HandleAsyncRequest(fmt.Sprintf("/v3/apps/%s", appGuid), "DELETE")
}

func DeleteIsolationSegment(guid string) {
	Eventually(cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments/%s", guid), "-X", "DELETE"), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func EntitleOrgToIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations", isoSegGuid),
		"-X",
		"POST",
		"-d",
		fmt.Sprintf(`{"data":[{ "guid":"%s" }]}`, orgGuid)),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func FetchRecentLogs(appGuid, oauthToken string, config config.CatsConfig) *Session {
	loggregatorEndpoint := getHttpLoggregatorEndpoint()
	logUrl := fmt.Sprintf("%s/apps/%s/recentlogs", loggregatorEndpoint, appGuid)
	session := helpers.Curl(Config, logUrl, "-H", fmt.Sprintf("Authorization: %s", oauthToken))
	Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	return session
}

func GetAuthToken() string {
	session := cf.Cf("oauth-token")
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return strings.TrimSpace(string(bytes))
}

func GetDefaultIsolationSegment(orgGuid string) string {
	session := cf.Cf("curl", fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return GetIsolationSegmentGuidFromResponse(bytes)
}

func GetDropletFromBuild(buildGuid string) string {
	buildPath := fmt.Sprintf("/v3/builds/%s", buildGuid)
	session := cf.Cf("curl", buildPath).Wait(Config.DefaultTimeoutDuration())
	var build struct {
		Droplet struct {
			Guid string `json:"guid"`
		} `json:"droplet"`
	}
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	json.Unmarshal(bytes, &build)
	return build.Droplet.Guid
}

func GetGuidFromResponse(response []byte) string {
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if len(GetResponse.Resources) == 0 {
		Fail("No guid found for response")
	}

	return GetResponse.Resources[0].Guid
}

func GetIsolationSegmentGuid(name string) string {
	session := cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return GetGuidFromResponse(bytes)
}

func GetIsolationSegmentGuidFromResponse(response []byte) string {
	type data struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Data data `json:"data"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if (data{}) == GetResponse.Data {
		return ""
	}

	return GetResponse.Data.Guid
}

func GetSpaceGuidFromName(spaceName string) string {
	session := cf.Cf("space", spaceName, "--guid")
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return strings.TrimSpace(string(bytes))
}

func IsolationSegmentExists(name string) bool {
	session := cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func OrgEntitledToIsolationSegment(orgGuid string, isoSegName string) bool {
	session := cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments?names=%s&organization_guids=%s", isoSegName, orgGuid))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()

	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func RevokeOrgEntitlementForIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations/%s", isoSegGuid, orgGuid),
		"-X",
		"DELETE",
	), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func ScaleProcess(appGuid, processType, memoryInMb string) {
	scalePath := fmt.Sprintf("/v3/apps/%s/processes/%s/actions/scale", appGuid, processType)
	scaleBody := fmt.Sprintf(`{"memory_in_mb":"%s"}`, memoryInMb)
	session := cf.Cf("curl", scalePath, "-X", "POST", "-d", scaleBody).Wait(Config.DefaultTimeoutDuration())
	Expect(session).To(Exit(0))
	result := session.Out.Contents()
	Expect(strings.Contains(string(result), "errors")).To(BeFalse())
}

func SendRequestWithSpoofedHeader(host, domain string) *http.Response {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://wildcard-path.%s", domain), nil)
	req.Host = host

	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}

func SetDefaultIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func StageBuildpackPackage(packageGuid string, buildpacks ...string) string {
	stageBody := fmt.Sprintf(`{"lifecycle":{ "type": "buildpack", "data": { "buildpacks": ["%s"] } }, "package": { "guid" : "%s"}}`, strings.Join(buildpacks, `", "`), packageGuid)
	stageUrl := "/v3/builds"
	session := cf.Cf("curl", stageUrl, "-X", "POST", "-d", stageBody)
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	var build struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &build)
	Expect(build.Guid).NotTo(BeEmpty())
	return build.Guid
}

func StageDockerPackage(packageGuid string) string {
	stageBody := fmt.Sprintf(`{"lifecycle": { "type" : "docker", "data": {} }, "package": { "guid" : "%s"}}`, packageGuid)
	stageUrl := "/v3/builds"
	session := cf.Cf("curl", stageUrl, "-X", "POST", "-d", stageBody)
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	var build struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &build)
	return build.Guid
}

func StartApp(appGuid string) {
	startURL := fmt.Sprintf("/v3/apps/%s/actions/start", appGuid)
	Expect(cf.Cf("curl", startURL, "-X", "POST").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
}

func StopApp(appGuid string) {
	stopURL := fmt.Sprintf("/v3/apps/%s/actions/stop", appGuid)
	Expect(cf.Cf("curl", stopURL, "-X", "POST").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
}

func UnassignIsolationSegmentFromSpace(spaceGuid string) {
	Eventually(cf.Cf("curl", fmt.Sprintf("/v3/spaces/%s/relationships/isolation_segment", spaceGuid),
		"-X",
		"PATCH",
		"-d",
		`{"data":{"guid":null}}`),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func UnsetDefaultIsolationSegment(orgGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
		"-X",
		"PATCH",
		"-d",
		`{"data":{"guid": null}}`),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func UploadPackage(uploadUrl, packageZipPath, token string) {
	bits := fmt.Sprintf(`bits=@%s`, packageZipPath)
	curl := helpers.Curl(Config, "-v", "-s", uploadUrl, "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
}

func WaitForBuildToStage(buildGuid string) {
	buildPath := fmt.Sprintf("/v3/builds/%s", buildGuid)
	Eventually(func() *Session {
		session := cf.Cf("curl", buildPath).Wait(Config.DefaultTimeoutDuration())
		Expect(session).NotTo(Say("FAILED"))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
}

func WaitForDropletToCopy(dropletGuid string) {
	dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGuid)
	Eventually(func() *Session {
		session := cf.Cf("curl", dropletPath).Wait(Config.DefaultTimeoutDuration())
		Expect(session).NotTo(Say("FAILED"))
		return session
	}, Config.CfPushTimeoutDuration()).Should(Say("STAGED"))
}

func WaitForPackageToBeReady(packageGuid string) {
	pkgUrl := fmt.Sprintf("/v3/packages/%s", packageGuid)
	Eventually(func() *Session {
		session := cf.Cf("curl", pkgUrl)
		Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		return session
	}, Config.LongCurlTimeoutDuration()).Should(Say("READY"))
}

//private

func getHttpLoggregatorEndpoint() string {
	infoCommand := cf.Cf("curl", "/v2/info")
	Expect(infoCommand.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

	var response struct {
		DopplerLoggingEndpoint string `json:"doppler_logging_endpoint"`
	}

	err := json.Unmarshal(infoCommand.Buffer().Contents(), &response)
	Expect(err).NotTo(HaveOccurred())

	return strings.Replace(response.DopplerLoggingEndpoint, "ws", "http", 1)
}
