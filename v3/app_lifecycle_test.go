package v3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

type Entity struct {
	AppName       string `json:"app_name"`
	AppGuid       string `json:"app_guid"`
	State         string `json:"state"`
	BuildpackName string `json:"buildpack_name"`
	BuildpackGuid string `json:"buildpack_guid"`
	ParentAppName string `json:"parent_app_name"`
	ParentAppGuid string `json:"parent_app_guid"`
	ProcessType   string `json:"process_type"`
}
type AppUsageEvent struct {
	Entity `json:"entity"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
}

func eventsInclude(events []AppUsageEvent, event AppUsageEvent) bool {
	found := false
	for _, e := range events {
		found = event.Entity.ParentAppName == e.Entity.ParentAppName &&
			event.Entity.ParentAppGuid == e.Entity.ParentAppGuid &&
			event.Entity.ProcessType == e.Entity.ProcessType &&
			event.Entity.State == e.Entity.State &&
			event.Entity.AppGuid == e.Entity.AppGuid
		if found {
			break
		}
	}
	return found
}

func lastAppUsageEvent(appName string, state string) (bool, AppUsageEvent) {
	var response AppUsageEvents
	cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		cf.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response, DEFAULT_TIMEOUT)
	})

	for _, event := range response.Resources {
		if event.Entity.AppName == appName && event.Entity.State == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

func lastPageUsageEvents(appName string) []AppUsageEvent {
	var response AppUsageEvents

	cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		cf.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response, DEFAULT_TIMEOUT)
	})

	return response.Resources
}

func stagePackage(packageGuid, stageBody string) string {
	stageUrl := fmt.Sprintf("/v3/packages/%s/droplets", packageGuid)
	session := cf.Cf("curl", stageUrl, "-X", "POST", "-d", stageBody)
	bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()
	var droplet struct {
		Guid string `json:"guid"`
	}
	json.Unmarshal(bytes, &droplet)
	return droplet.Guid
}

type ProcessList struct {
	Processes []Process `json:"resources"`
}

type Process struct {
	Guid    string `json:"guid"`
	Type    string `json:"type"`
	Command string `json:"command"`

	Name string `json:"-"`
}

func getProcess(appGuid, appName string) []Process {
	processesURL := fmt.Sprintf("/v3/apps/%s/processes", appGuid)
	session := cf.Cf("curl", processesURL)
	bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()

	processes := ProcessList{}
	json.Unmarshal(bytes, &processes)

	for i, process := range processes.Processes {
		processes.Processes[i].Name = fmt.Sprintf("v3-%s-%s", appName, process.Type)
	}

	return processes.Processes
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

mkdir -p $1 $2
if [ -f "$2/cached-file" ]; then
	cp $2/cached-file $1/content
else
	echo "cache not found" > $1/content
fi

content=$(cat $1/content)
echo "web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "custom buildpack contents - $content"; } | nc -l \$PORT; done" > $1/Procfile

echo "here's a cache" > $2/cached-file
`,
			},
			{
				Name: "bin/detect",
				Body: `#!/bin/bash
echo no
exit 1
`,
			},
			{
				Name: "bin/release",
				Body: `#!/usr/bin/env bash


cat <<EOF
---
config_vars:
  PATH: bin:/usr/local/bin:/usr/bin:/bin
  FROM_BUILD_PACK: "yes"
default_process_types:
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "custom buildpack contents - $content"; } | nc -l \$PORT; done
EOF
`,
			},
		})

		return buildpackArchivePath
	}

	var (
		appName               string
		appGuid               string
		buildpackName         string
		buildpackGuid         string
		packageGuid           string
		spaceGuid             string
		token                 string
		environment_variables string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")

		buildpackName = generator.RandomName()
		buildpackZip := createBuildpack()

		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		session := cf.Cf("curl", fmt.Sprintf("/v2/spaces?q=name:%s", context.RegularUserContext().Space))
		bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()
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
		bytes = session.Wait(DEFAULT_TIMEOUT).Out.Contents()
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
		environment_variables = `{"foo":"bar"}`
		session = cf.Cf("curl", "/v3/apps", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s", "space_guid":"%s", "environment_variables":%s}`, appName, spaceGuid, environment_variables))
		bytes = session.Wait(DEFAULT_TIMEOUT).Out.Contents()
		var app struct {
			Guid string `json:"guid"`
		}
		json.Unmarshal(bytes, &app)
		appGuid = app.Guid

		// CREATE PACKAGE
		packageCreateUrl := fmt.Sprintf("/v3/apps/%s/packages", appGuid)
		session = cf.Cf("curl", packageCreateUrl, "-X", "POST", "-d", fmt.Sprintf(`{"type":"bits"}`))
		bytes = session.Wait(DEFAULT_TIMEOUT).Out.Contents()
		var pac struct {
			Guid string `json:"guid"`
		}
		json.Unmarshal(bytes, &pac)
		packageGuid = pac.Guid

		// UPLOAD PACKAGE
		bytes = runner.Run("bash", "-c", "cf oauth-token | tail -n +4").Wait(DEFAULT_TIMEOUT).Out.Contents()
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
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	})

	XIt("Stages with a user specified admin buildpack", func() {
		dropletGuid := stagePackage(packageGuid, fmt.Sprintf(`{"buildpack":"%s"}`, buildpackName))

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session := runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return session
		}, 1*time.Minute, 10*time.Second).Should(Say("STAGED WITH CUSTOM BUILDPACK"))
		Expect("staging output").To(Say("SOME_ENV_VAR=foo"))
	})

	XIt("Stages with a user specified github buildpack", func() {
		dropletGuid := stagePackage(packageGuid, `{"buildpack":"http://github.com/cloudfoundry/go-buildpack"}`)

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session := runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			fmt.Println(string(session.Out.Contents()))
			return session
		}, 3*time.Minute, 10*time.Second).Should(Say("Cloning into"))
	})

	It("uses buildpack cache for staging", func() {
		firstDropletGuid := stagePackage(packageGuid, fmt.Sprintf(`{"buildpack":"%s"}`, buildpackName))
		dropletPath := fmt.Sprintf("/v3/droplets/%s", firstDropletGuid)
		Eventually(func() *Session {
			result := cf.Cf("curl", dropletPath).Wait(DEFAULT_TIMEOUT)
			if strings.Contains(string(result.Out.Contents()), "FAILED") {
				Fail("staging failed")
			}
			return result
		}, CF_PUSH_TIMEOUT).Should(Say("custom buildpack contents - cache not found"))

		// Wait for buildpack cache to be uploaded to blobstore.
		time.Sleep(DEFAULT_TIMEOUT)

		secondDropletGuid := stagePackage(packageGuid, fmt.Sprintf(`{"buildpack":"%s"}`, buildpackName))
		dropletPath = fmt.Sprintf("/v3/droplets/%s", secondDropletGuid)
		Eventually(func() *Session {
			result := cf.Cf("curl", dropletPath).Wait(DEFAULT_TIMEOUT)
			if strings.Contains(string(result.Out.Contents()), "FAILED") {
				Fail("staging failed")
			}
			if strings.Contains(string(result.Out.Contents()), "cache not found") {
				Fail("cache was not found")
			}
			return result
		}, CF_PUSH_TIMEOUT).Should(Say("custom buildpack contents - here's a cache"))

		Expect(secondDropletGuid).NotTo(Equal(firstDropletGuid))
	})

	It("can run apps", func() {
		dropletGuid := stagePackage(packageGuid, "{}")
		dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGuid)
		Eventually(func() *Session {
			return cf.Cf("curl", dropletPath).Wait(DEFAULT_TIMEOUT)
		}, CF_PUSH_TIMEOUT).Should(Say("STAGED"))

		appUpdatePath := fmt.Sprintf("/v3/apps/%s/current_droplet", appGuid)
		appUpdateBody := fmt.Sprintf(`{"droplet_guid":"%s"}`, dropletGuid)
		Expect(cf.Cf("curl", appUpdatePath, "-X", "PUT", "-d", appUpdateBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		var webProcess Process
		var workerProcess Process
		processes := getProcess(appGuid, appName)
		for _, process := range processes {
			if process.Type == "web" {
				webProcess = process
			} else if process.Type == "worker" {
				workerProcess = process
			}
		}

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

		startApp(appGuid)

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

		output := helpers.CurlApp(webProcess.Name, "/env")
		Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
		Expect(output).To(ContainSubstring(`"foo"=>"bar"`))

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))
		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", workerProcess.Name)))

		usageEvents := lastPageUsageEvents(appName)

		event1 := AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		event2 := AppUsageEvent{Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(eventsInclude(usageEvents, event1)).To(BeTrue())
		Expect(eventsInclude(usageEvents, event2)).To(BeTrue())

		stopApp(appGuid)

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", webProcess.Name)))
		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", workerProcess.Name)))

		usageEvents = lastPageUsageEvents(appName)
		event1 = AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		event2 = AppUsageEvent{Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(eventsInclude(usageEvents, event1)).To(BeTrue())
		Expect(eventsInclude(usageEvents, event2)).To(BeTrue())

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
	})

	XIt("can download package bits", func() {
		var out bytes.Buffer

		tmpdir, err := ioutil.TempDir(os.TempDir(), "package-download")
		Expect(err).ToNot(HaveOccurred())

		app_package_path := path.Join(tmpdir, appName)

		session := cf.Cf("curl", fmt.Sprintf("/v3/packages/%s/download", packageGuid), "--output", app_package_path).Wait(DEFAULT_TIMEOUT)
		Expect(session).To(Exit(0))

		cmd := exec.Command("unzip", app_package_path)
		cmd.Stdout = &out
		err = cmd.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(out.String()).To(ContainSubstring("dora.rb"))
	})
})
