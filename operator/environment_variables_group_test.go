package operator

import (
	"io/ioutil"
	"path"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

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
sleep 5
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

	var originalRunningEnv string
	var originalStagingEnv string
	var appName string
	var buildpackName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			session := cf.Cf("curl", "/v2/config/environment_variable_groups/running").Wait(DEFAULT_TIMEOUT)
			Expect(session).To(Exit(0))
			originalRunningEnv = string(session.Out.Contents())

			session = cf.Cf("curl", "/v2/config/environment_variable_groups/staging").Wait(DEFAULT_TIMEOUT)
			Expect(session).To(Exit(0))
			originalStagingEnv = string(session.Out.Contents())
		})
	})

	AfterEach(func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("curl", "/v2/config/environment_variable_groups/staging", "-X", "PUT", "-d", originalStagingEnv).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("curl", "/v2/config/environment_variable_groups/running", "-X", "PUT", "-d", originalRunningEnv).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			if buildpackName != "" {
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			}
		})
		Expect(cf.Cf("delete", appName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	It("Applies correct environment variables while running apps", func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("set-running-environment-variable-group", `{"CATS_RUNNING_TEST_VAR":"running_env_value"}`).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		Expect(cf.Cf("push", appName, "-m", "128M", "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		env := helpers.CurlApp(appName, "/env")

		Expect(env).To(ContainSubstring("running_env_value"))
	})

	It("Applies environment variables while staging apps", func() {
		buildpackName = generator.RandomName()
		buildpackZip := createBuildpack()

		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("set-staging-environment-variable-group", `{"CATS_STAGING_TEST_VAR":"staging_env_value"}`).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		Expect(cf.Cf("push", appName, "-m", "128M", "-b", buildpackName, "-p", assets.NewAssets().HelloWorld, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(1))

		Eventually(func() *Session {
			appLogsSession := cf.Cf("logs", "--recent", appName)
			Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			return appLogsSession
		}, DEFAULT_TIMEOUT).Should(Say("staging_env_value"))
	})
})
