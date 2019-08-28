package v3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = V3Describe("buildpack", func() {
	var (
		appName       string
		appGuid       string
		buildpackName string
		packageGuid   string
		spaceGuid     string
		token         string
	)

	type buildpack struct {
		Name          string `json:"name"`
		DetectOutput  string `json:"detect_output"`
		BuildpackName string `json:"buildpack_name"`
		Version       string `json:"version"`
	}

	Context("With a single buildpack app", func() {
		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
			appGuid = CreateApp(appName, spaceGuid, "{}")
			packageGuid = CreatePackage(appGuid)

			token = GetAuthToken()
			uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)
			UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
			WaitForPackageToBeReady(packageGuid)

			buildpackName = random_name.CATSRandomName("BPK")
			buildpackZip := createBuildpack()

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait()).To(Exit(0))
			})
		})

		AfterEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait()).To(Exit(0))
			})

			app_helpers.AppReport(appName)
			DeleteApp(appGuid)
		})

		It("Stages with a user specified admin buildpack", func() {
			StageBuildpackPackage(packageGuid, buildpackName)
			Eventually(func() *Session {
				return FetchRecentLogs(appGuid, token, Config)
			}).Should(Say("STAGED WITH CUSTOM BUILDPACK"))
		})

		It("Downloads the correct user specified git buildpack", func() {
			if !Config.GetIncludeInternetDependent() {
				Skip(skip_messages.SkipInternetDependentMessage)
			}
			StageBuildpackPackage(packageGuid, "https://github.com/cloudfoundry/example-git-buildpack")

			Eventually(func() *Session {
				return FetchRecentLogs(appGuid, token, Config)
			}).Should(Say("I'm a buildpack!"))
		})

		It("uses buildpack cache for staging", func() {
			firstBuildGuid := StageBuildpackPackage(packageGuid, buildpackName)
			WaitForBuildToStage(firstBuildGuid)
			dropletGuid := GetDropletFromBuild(firstBuildGuid)
			dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGuid)

			result := cf.Cf("curl", dropletPath).Wait()
			Expect(result).To(Say("custom buildpack contents - cache not found"))

			// Wait for buildpack cache to be uploaded to blobstore.
			time.Sleep(Config.SleepTimeoutDuration())

			secondBuildGuid := StageBuildpackPackage(packageGuid, buildpackName)
			WaitForBuildToStage(secondBuildGuid)
			dropletGuid = GetDropletFromBuild(secondBuildGuid)
			dropletPath = fmt.Sprintf("/v3/droplets/%s", dropletGuid)
			result = cf.Cf("curl", dropletPath).Wait()
			Expect(result).To(Say("custom buildpack contents - here's a cache"))

			Expect(secondBuildGuid).NotTo(Equal(firstBuildGuid))
		})
	})

	Context("With a multi buildpack app", func() {
		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
			appGuid = CreateApp(appName, spaceGuid, `{"GOPACKAGENAME": "go-online"}`)
			packageGuid = CreatePackage(appGuid)

			token = GetAuthToken()
			uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)
			UploadPackage(uploadURL, assets.NewAssets().GoCallsRubyZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			DeleteApp(appGuid)
		})

		It("Stages with multiple admin buildpacks", func() {
			buildGUID := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName(), Config.GetGoBuildpackName())
			WaitForBuildToStage(buildGUID)

			dropletGUID := GetDropletFromBuild(buildGUID)
			var droplet struct {
				Buildpacks []buildpack `json:"buildpacks"`
			}
			dropletPath := fmt.Sprintf("/v3/droplets/%s", dropletGUID)
			session := cf.Cf("curl", dropletPath)
			bytes := session.Wait().Out.Contents()
			json.Unmarshal(bytes, &droplet)
			Expect(len(droplet.Buildpacks)).To(Equal(2))
			i := 0
			j := 1
			if droplet.Buildpacks[0].Name == "go_buildpack" {
				i, j = j, i
			}
			Expect(droplet.Buildpacks[i].Name).To(Equal("ruby_buildpack"))
			Expect(droplet.Buildpacks[i].DetectOutput).To(Equal(""))
			Expect(droplet.Buildpacks[i].BuildpackName).To(Equal("ruby"))
			Expect(droplet.Buildpacks[i].Version).ToNot(BeEmpty())
			Expect(droplet.Buildpacks[j].Name).To(Equal("go_buildpack"))
			Expect(droplet.Buildpacks[j].DetectOutput).To(Equal("go"))
			Expect(droplet.Buildpacks[j].BuildpackName).To(Equal("go"))
			Expect(droplet.Buildpacks[j].Version).ToNot(BeEmpty())

			AssignDropletToApp(appGuid, dropletGUID)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")

			Expect(webProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(appGuid, TestSetup.RegularUserContext().Space, Config.GetAppsDomain(), webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}).Should(ContainSubstring("The bundler version is"))
		})
	})
})

func createBuildpack() string {
	tmpPath, err := ioutil.TempDir("", "buildpack-cats")
	Expect(err).ToNot(HaveOccurred())

	buildpackArchivePath := path.Join(tmpPath, "buildpack.zip")

	archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
		{
			Name: "bin/compile",
			Body: `#!/usr/bin/env bash

echo "STAGED WITH CUSTOM BUILDPACK"

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
