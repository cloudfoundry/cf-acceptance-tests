package v3

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
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
			Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})
		DeleteApp(appGuid)
	})

	It("Stages with a user specified admin buildpack", func() {
		StageBuildpackPackage(packageGuid, buildpackName)
		Eventually(func() *Session {
			return FetchRecentLogs(appGuid, token, Config)
		}, 1*time.Minute, 10*time.Second).Should(Say("STAGED WITH CUSTOM BUILDPACK"))
	})

	It("Downloads the correct user specified git buildpack", func() {
		if !Config.GetIncludeInternetDependent() {
			Skip(skip_messages.SkipInternetDependentMessage)
		}
		StageBuildpackPackage(packageGuid, "https://github.com/cloudfoundry/example-git-buildpack")

		Eventually(func() *Session {
			return FetchRecentLogs(appGuid, token, Config)
		}, 3*time.Minute, 10*time.Second).Should(Say("I'm a buildpack!"))
	})

	It("uses buildpack cache for staging", func() {
		firstDropletGuid := StageBuildpackPackage(packageGuid, buildpackName)
		dropletPath := fmt.Sprintf("/v3/droplets/%s", firstDropletGuid)
		Eventually(func() *Session {
			result := cf.Cf("curl", dropletPath).Wait(Config.DefaultTimeoutDuration())
			if strings.Contains(string(result.Out.Contents()), "FAILED") {
				Fail("staging failed")
			}
			return result
		}, Config.CfPushTimeoutDuration()).Should(Say("custom buildpack contents - cache not found"))

		// Wait for buildpack cache to be uploaded to blobstore.
		time.Sleep(Config.SleepTimeoutDuration())

		secondDropletGuid := StageBuildpackPackage(packageGuid, buildpackName)
		dropletPath = fmt.Sprintf("/v3/droplets/%s", secondDropletGuid)
		Eventually(func() *Session {
			result := cf.Cf("curl", dropletPath).Wait(Config.DefaultTimeoutDuration())
			if strings.Contains(string(result.Out.Contents()), "FAILED") {
				Fail("staging failed")
			}
			if strings.Contains(string(result.Out.Contents()), "cache not found") {
				Fail("cache was not found")
			}
			return result
		}, Config.CfPushTimeoutDuration()).Should(Say("custom buildpack contents - here's a cache"))

		Expect(secondDropletGuid).NotTo(Equal(firstDropletGuid))
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
