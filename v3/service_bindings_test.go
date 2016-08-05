package v3

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
)

var _ = Describe("service bindings", func() {
	var (
		appName     string
		spaceGuid   string
		appGuid     string
		packageGuid string
		token       string
		upsName     string
		upsGuid     string
	)

	BeforeEach(func() {
		appName = generator.RandomNameForResource("APP")
		upsName = generator.RandomNameForResource("SVC")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, "{}")
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", config.Protocol(), config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
		Expect(cf.Cf("create-user-provided-service", upsName, "-p", "{\"username\":\"admin\",\"password\":\"my-service\"}").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		session := cf.Cf("service", upsName, "--guid")
		Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		upsGuid = strings.Trim(string(session.Out.Contents()), "\n")

		Expect(cf.Cf("curl", "/v3/service_bindings", "-X", "POST", "-d", fmt.Sprintf(`
		{
			"type": "app",
			"relationships": {
			  "app": { "guid": "%s" },
			  "service_instance": { "guid": "%s" }
			}
		}`, appGuid, upsGuid)).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, config)
		DeleteApp(appGuid)
		Expect(cf.Cf("delete-service", upsName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("staging", func() {
		var buildpackName string

		BeforeEach(func() {
			buildpackName = generator.RandomNameForResource("BPK")
			buildpackZip := createEnvBuildpack()
			cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})

		AfterEach(func() {
			cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})

		// TODO Unpend this test once v3 service bindings can be deleted (especially recursively through org delete)
		PIt("exposes them during staging", func() {
			StageBuildpackPackage(packageGuid, buildpackName)
			Eventually(func() *Session {
				return FetchRecentLogs(appGuid, token, config)
			}, 1*time.Minute, 10*time.Second).Should(Say("my-service"))
		})
	})

	// TODO Unpend this test once v3 service bindings can be deleted (especially recursively through org delete)
	PIt("exposes them during running", func() {
		dropletGuid := StageBuildpackPackage(packageGuid, config.RubyBuildpackName)
		WaitForDropletToStage(dropletGuid)
		AssignDropletToApp(appGuid, dropletGuid)
		CreateAndMapRoute(appGuid, context.RegularUserContext().Space, config.AppsDomain, appName)

		StartApp(appGuid)

		Eventually(func() string {
			return helpers.CurlApp(appName, "/env")
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("my-service"))
	})
})

func createEnvBuildpack() string {
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

env

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
