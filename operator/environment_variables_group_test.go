package operator

import (
	"io/ioutil"
	"path"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("Environment Variables Groups", func() {
	var createBuildpack = func() string {
		tmpPath, err := ioutil.TempDir("", "env-group-staging")
		Expect(err).ToNot(HaveOccurred())

		buildpackArchivePath := path.Join(tmpPath, "buildpack.zip")

		archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
			{
				Name: "bin/compile",
				Body: `#!/usr/bin/env bash
echo $CATS_STAGING_TEST_VAR
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

	It("Applies correct environment variables while running apps", func() {
		var originalEnv string
		appName := generator.RandomName()

		cf.AsUser(context.AdminUserContext(), func() {
			session := cf.Cf("curl", "/v2/config/environment_variable_groups/staging").Wait(DEFAULT_TIMEOUT)
			Expect(session).To(Exit(0))
			originalEnv = string(session.Out.Contents())

			Expect(cf.Cf("curl", "/v2/config/environment_variable_groups/running", "-X", "PUT", "-d", `{"CATS_RUNNING_TEST_VAR":"running_env_value"}`).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		defer func() {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("curl", "/v2/config/environment_variable_groups/staging", "-X", "PUT", "-d", originalEnv).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		}()

		Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		defer func() { cf.Cf("delete", appName, "-f").Wait(CF_PUSH_TIMEOUT) }()

		env := helpers.CurlApp(appName, "/env")

		Expect(env).To(ContainSubstring("running_env_value"))
	})

	It("Applies environment variables while staging apps", func() {
		var originalEnv string
		appName := generator.RandomName()
		buildpackName := generator.RandomName()
		buildpackZip := createBuildpack()

		cf.AsUser(context.AdminUserContext(), func() {
			session := cf.Cf("curl", "/v2/config/environment_variable_groups/staging").Wait(DEFAULT_TIMEOUT)
			Expect(session).To(Exit(0))
			originalEnv = string(session.Out.Contents())

			Expect(cf.Cf("curl", "/v2/config/environment_variable_groups/staging", "-X", "PUT", "-d", `{"CATS_STAGING_TEST_VAR":"staging_env_value"}`).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		defer func() {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("curl", "/v2/config/environment_variable_groups/staging", "-X", "PUT", "-d", originalEnv).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		}()

		Expect(cf.Cf("push", appName, "-b", buildpackName, "-p", helpers.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))
		defer func() { cf.Cf("delete", appName, "-f").Wait(CF_PUSH_TIMEOUT) }()

		Eventually(func() *Session {
			appLogsSession := cf.Cf("logs", "--recent", appName)
			Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return appLogsSession
		}, 5).Should(Say("staging_env_value"))
	})
})
