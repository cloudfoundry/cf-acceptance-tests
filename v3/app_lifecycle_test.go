package v3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"os/exec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

type AppUsageEvent struct {
	Entity struct {
		AppName       string `json:"app_name"`
		State         string `json:"state"`
		BuildpackName string `json:"buildpack_name"`
		BuildpackGuid string `json:"buildpack_guid"`
	} `json:"entity"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
}

func lastAppUsageEvent(appName string, state string) (bool, AppUsageEvent) {
	var response AppUsageEvents
	cf.AsUser(context.AdminUserContext(), func() {
		cf.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response)
	})

	for _, event := range response.Resources {
		if event.Entity.AppName == appName && event.Entity.State == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

func stagePackage(packageGuid, stageBody string) string {
	stageUrl := fmt.Sprintf("/v3/packages/%s/droplets", packageGuid)
	session := cf.Cf("curl", stageUrl, "-X", "POST", "-d", stageBody)
	bytes := session.Wait().Out.Contents()
	var droplet struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &droplet)
	return droplet.Guid
}

type Process struct {
	Guid    string `json:"guid"`
	Type    string `json:"type"`
	Command string `json:"command"`

	Name string `json:"-"`
}

func addProcess(appGuid, processType, spaceGuid string) Process {
	processBody := fmt.Sprintf(`{"type":"%s","space_guid":"%s"}`, processType, spaceGuid)
	session := cf.Cf("curl", "/v3/processes", "-X", "POST", "-d", processBody)
	bytes := session.Wait().Out.Contents()
	var process Process
	json.Unmarshal(bytes, &process)
	addProcessURL := fmt.Sprintf("/v3/apps/%s/processes", appGuid)
	addProcessBody := fmt.Sprintf(`{"process_guid": "%s"}`, process.Guid)
	Expect(cf.Cf("curl", addProcessURL, "-X", "PUT", "-d", addProcessBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	process.Name = fmt.Sprintf("v3-proc-%s-%s", process.Type, process.Guid)
	return process
}

func startApp(appGuid string) {
	startURL := fmt.Sprintf("/v3/apps/%s/start", appGuid)
	Expect(cf.Cf("curl", startURL, "-X", "PUT").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func stopApp(appGuid string) {
	stopURL := fmt.Sprintf("/v3/apps/%s/stop", appGuid)
	Expect(cf.Cf("curl", stopURL, "-X", "PUT").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

var _ = Describe("v3 app lifecycle", func() {
	var createBuildpack = func() string {
		tmpPath, err := ioutil.TempDir("", "env-group-staging")
		Expect(err).ToNot(HaveOccurred())

		buildpackArchivePath := path.Join(tmpPath, "buildpack.zip")

		archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
			{
				Name: "bin/compile",
				Body: `#!/usr/bin/env bash
sleep 5
echo "STAGED WITH CUSTOM BUILDPACK"
exit 1
`,
			},
			{
				Name: "bin/detect",
				Body: `#!/bin/bash
exit 1
`,
			},
			{
				Name: "bin/release",
				Body: `#!/usr/bin/env bash
exit 1
`,
			},
		})

		return buildpackArchivePath
	}

	var (
		appName       string
		appGuid       string
		buildpackName string
		buildpackGuid string
		packageGuid   string
		spaceGuid     string
		token         string
	)

	BeforeEach(func() {
		appName = generator.RandomName()

		buildpackName = generator.RandomName()
		buildpackZip := createBuildpack()

		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		session := cf.Cf("curl", fmt.Sprintf("/v2/spaces?q=name:%s", context.RegularUserContext().Space))
		bytes := session.Wait().Out.Contents()
		var space struct {
			Resources []struct {
				Metadata struct {
					Guid string `json:"guid"`
				} `json:"metadata"`
			} `json:"resources"`
		}
		json.Unmarshal(bytes, &space)
		spaceGuid = space.Resources[0].Metadata.Guid

		session = cf.Cf("curl", fmt.Sprintf("/v2/buildpacks?q=name:%s", buildpackName))
		bytes = session.Wait().Out.Contents()
		var buildpack struct {
			Resources []struct {
				Metadata struct {
					Guid string `json:"guid"`
				} `json:"metadata"`
			} `json:"resources"`
		}
		json.Unmarshal(bytes, &buildpack)
		buildpackGuid = buildpack.Resources[0].Metadata.Guid

		// CREATE APP
		session = cf.Cf("curl", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "space_guid":"%s"}`, appName, spaceGuid))
		bytes = session.Wait().Out.Contents()
		var app struct {
			Guid string `json:"guid"`
		}
		json.Unmarshal(bytes, &app)
		appGuid = app.Guid

		// CREATE PACKAGE
		packageCreateUrl := fmt.Sprintf("/v3/apps/%s/packages", appGuid)
		session = cf.Cf("curl", packageCreateUrl, "-X", "POST", "-d", fmt.Sprintf(`{"type":"bits"}`))
		bytes = session.Wait().Out.Contents()
		var pac struct {
			Guid string `json:"guid"`
		}
		json.Unmarshal(bytes, &pac)
		packageGuid = pac.Guid

		// UPLOAD PACKAGE
		bytes = runner.Run("bash", "-c", "cf oauth-token | tail -n +4").Wait(5).Out.Contents()
		token = strings.TrimSpace(string(bytes))
		doraZip := fmt.Sprintf(`bits=@"%s"`, assets.NewAssets().DoraZip)
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)

		_, err := exec.Command("curl", "-v", "-s", uploadUrl, "-F", doraZip, "-H", fmt.Sprintf("Authorization: %s", token)).CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		pkgUrl := fmt.Sprintf("/v3/packages/%s", packageGuid)
		Eventually(func() *Session {
			session = cf.Cf("curl", pkgUrl)
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return session
		}, 1*time.Minute).Should(Say("READY"))
	})

	AfterEach(func() {
		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	})

	XIt("Stages with a user specified admin buildpack", func() {
		dropletGuid := stagePackage(packageGuid, fmt.Sprintf(`{"buildpack_guid":"%s"}`, buildpackGuid))

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session := runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return session
		}, 1*time.Minute, 10*time.Second).Should(Say("STAGED WITH CUSTOM BUILDPACK"))
	})

	XIt("Stages with a user specified github buildpack", func() {
		dropletGuid := stagePackage(packageGuid, `{"buildpack_git_url":"http://github.com/cloudfoundry/go-buildpack"}`)

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session := runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			fmt.Println(string(session.Out.Contents()))
			return session
		}, 3*time.Minute, 10*time.Second).Should(Say("Cloning into"))
	})

	It("can run apps", func() {
		dropletGuid := stagePackage(packageGuid, "{}")
		dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGuid)
		Eventually(func() *Session {
			return cf.Cf("curl", dropletPath).Wait(DEFAULT_TIMEOUT)
		}, CF_PUSH_TIMEOUT).Should(Say("STAGED"))

		webProcess := addProcess(appGuid, "web", spaceGuid)
		workerProcess := addProcess(appGuid, "worker", spaceGuid)

		Expect(cf.Cf("create-route", context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, "-n", webProcess.Name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		getRoutePath := fmt.Sprintf("/v2/routes?q=host:%s", webProcess.Name)
		routeBody := cf.Cf("curl", getRoutePath).Wait(DEFAULT_TIMEOUT).Out.Contents()
		routeJSON := struct {
			Resources []struct {
				Metadata struct {
					Guid string `json:"guid"`
				} `json:"metadata"`
			} `json:"resources"`
		}{}
		json.Unmarshal([]byte(routeBody), &routeJSON)
		routeGuid := routeJSON.Resources[0].Metadata.Guid
		addRoutePath := fmt.Sprintf("/v3/apps/%s/routes", appGuid)
		addRouteBody := fmt.Sprintf(`{"route_guid":"%s"}`, routeGuid)
		Expect(cf.Cf("curl", addRoutePath, "-X", "PUT", "-d", addRouteBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		appUpdatePath := fmt.Sprintf("/v3/apps/%s", appGuid)
		appUpdateBody := fmt.Sprintf(`{"desired_droplet_guid":"%s"}`, dropletGuid)
		Expect(cf.Cf("curl", appUpdatePath, "-X", "PATCH", "-d", appUpdateBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		startApp(appGuid)

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))
		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", workerProcess.Name)))

		stopApp(appGuid)

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", webProcess.Name)))
		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", workerProcess.Name)))

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
	})
})
