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
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("v3 staging", func() {
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

	var appName string
	var appGuid string
	var buildpackName string
	var buildpackGuid string
	var packageGuid string
	var spaceGuid string
	var token string

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
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
		bytes, _ = exec.Command("curl", "-v", "-s", uploadUrl, "-F", `bits=@"/Users/pivotal/workspace/cf-release/src/acceptance-tests/dora.zip"`, "-H", fmt.Sprintf("Authorization: %s", token)).CombinedOutput()
		pkgUrl := fmt.Sprintf("/v3/packages/%s", packageGuid)
		Eventually(func() *Session {
				session = cf.Cf("curl", pkgUrl)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				return session
			}, 1 * time.Minute).Should(Say("READY"))
	})

	AfterEach(func() {
		cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	})

	It("Stages with a user specified admin buildpack", func() {
		// STAGE PACKAGE
		stageUrl := fmt.Sprintf("/v3/packages/%s/droplets", packageGuid)
		session := cf.Cf("curl", stageUrl, "-X", "POST", "-d", fmt.Sprintf(`{"buildpack_guid":"%s"}`, buildpackGuid))
		bytes := session.Wait().Out.Contents()
		var droplet struct {
				Guid string `json:"guid"`
			}
		json.Unmarshal(bytes, &droplet)
		dropletGuid := droplet.Guid
		fmt.Println(string(bytes))

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
			session = runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
			Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return session
		}, 1 * time.Minute, 10 * time.Second).Should(Say("STAGED WITH CUSTOM BUILDPACK"))
	})

	It("Stages with a user specified admin buildpack", func() {
		// STAGE PACKAGE
		stageUrl := fmt.Sprintf("/v3/packages/%s/droplets", packageGuid)
		session := cf.Cf("curl", stageUrl, "-X", "POST", "-d", `{"buildpack_git_url":"http://github.com/cloudfoundry/go-buildpack"}`)
		bytes := session.Wait().Out.Contents()
		var droplet struct {
				Guid string `json:"guid"`
			}
		json.Unmarshal(bytes, &droplet)
		dropletGuid := droplet.Guid
		fmt.Println(string(bytes))

		logUrl := fmt.Sprintf("loggregator.%s/recent?app=%s", config.AppsDomain, dropletGuid)
		Eventually(func() *Session {
				session = runner.Curl(logUrl, "-H", fmt.Sprintf("Authorization: %s", token))
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				fmt.Println(string(session.Out.Contents()))
				return session
			}, 3 * time.Minute, 10 * time.Second).Should(Say("Cloning into"))
	})
})
