package v3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("buildpack", func() {
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
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, "{}")
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)

		buildpackName = generator.PrefixedRandomName("CATS-BP-")
		buildpackZip := createBuildpack()

		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		session := cf.Cf("curl", fmt.Sprintf("/v2/buildpacks?q=name:%s", buildpackName))
		bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()
		var buildpack struct {
			Resources []struct {
				Metadata struct {
					Guid string `json:"guid"`
				} `json:"metadata"`
			} `json:"resources"`
		}
		json.Unmarshal(bytes, &buildpack)
		buildpackGuid = buildpack.Resources[0].Metadata.Guid
	})

	AfterEach(func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	})

	XIt("Stages with a user specified admin buildpack", func() {
		dropletGuid := StagePackage(packageGuid, fmt.Sprintf(`{"buildpack":"%s"}`, buildpackName))

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session := runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return session
		}, 1*time.Minute, 10*time.Second).Should(Say("STAGED WITH CUSTOM BUILDPACK"))
	})

	XIt("Stages with a user specified github buildpack", func() {
		dropletGuid := StagePackage(packageGuid, `{"buildpack":"http://github.com/cloudfoundry/go-buildpack"}`)

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session := runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			fmt.Println(string(session.Out.Contents()))
			return session
		}, 3*time.Minute, 10*time.Second).Should(Say("Cloning into"))
	})

	It("uses buildpack cache for staging", func() {
		firstDropletGuid := StagePackage(packageGuid, fmt.Sprintf(`{"buildpack":"%s"}`, buildpackName))
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

		secondDropletGuid := StagePackage(packageGuid, fmt.Sprintf(`{"buildpack":"%s"}`, buildpackName))
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
