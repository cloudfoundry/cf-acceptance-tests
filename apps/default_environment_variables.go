package apps

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
)

var _ = AppsDescribe("Default Environment Variables", func() {
	var createBuildpack = func() string {
		tmpPath, err := os.MkdirTemp("", "default-env-var-test")
		Expect(err).ToNot(HaveOccurred())

		buildpackArchivePath := path.Join(tmpPath, "buildpack.zip")

		archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
			{
				Name: "bin/compile",
				Body: `#!/usr/bin/env bash
env
# wait for the log lines to make it through
sleep 5
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

	Describe("Default staging environment variables", func() {
		var appName string
		var buildpackName string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				if buildpackName != "" {
					Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait()).To(Exit(0))
				}
			})

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("applies default environment variables while staging apps", func() {
			buildpackName = random_name.CATSRandomName("BPK")
			buildpackZip := createBuildpack()

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait()).To(Exit(0))
			})

			pushSession := cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().HelloWorld,
				"-b", buildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
			).Wait(Config.CfPushTimeoutDuration())

			Eventually(pushSession, Config.CfPushTimeoutDuration()).Should(Exit(1))
			pushSessionContent := pushSession.Buffer().Contents()

			Expect(string(pushSessionContent)).To(MatchRegexp("LANG=en_US\\.UTF-8"))
			Expect(string(pushSessionContent)).To(MatchRegexp("CF_INSTANCE_INTERNAL_IP=.*"))
			Expect(string(pushSessionContent)).To(MatchRegexp("CF_INSTANCE_IP=.*"))
			Expect(string(pushSessionContent)).To(MatchRegexp("CF_INSTANCE_PORTS=.*"))
			Expect(string(pushSessionContent)).To(MatchRegexp("CF_STACK=.*"))
			Expect(string(pushSessionContent)).To(MatchRegexp("VCAP_APPLICATION=.*"))
			Expect(string(pushSessionContent)).To(MatchRegexp("VCAP_SERVICES=.*"))

			// these vars are set to the empty string (use m flag to make $ match eol)
			Expect(string(pushSessionContent)).To(MatchRegexp("(?m)CF_INSTANCE_ADDR=$"))
			Expect(string(pushSessionContent)).To(MatchRegexp("(?m)CF_INSTANCE_PORT=$"))
		})

		It("applies default environment variables while running apps and tasks", func() {
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
			),
				Config.CfPushTimeoutDuration(),
			).Should(Exit(0))

			envLines := helpers.CurlApp(Config, appName, "/env")

			var env map[string]string
			err := json.Unmarshal([]byte(envLines), &env)
			Expect(err).To(BeNil())

			Expect(env["LANG"]).To(Equal("en_US.UTF-8"))
			assertJsonParseable(env, "VCAP_APPLICATION", "VCAP_SERVICES")
			assertPresent(env,
				"CF_INSTANCE_GUID",
				"CF_INSTANCE_INDEX",
				"CF_INSTANCE_INTERNAL_IP",
				"CF_INSTANCE_IP",
				"CF_INSTANCE_PORTS",
				"VCAP_APP_HOST",
				"VCAP_APP_PORT",
			)

			if v, ok := env["CF_INSTANCE_PORT"]; ok && v != "" {
				Expect(v).To(MatchRegexp(`[0-9]+`))
			}
			if v, ok := env["CF_INSTANCE_ADDR"]; ok && v != "" {
				Expect(v).To(MatchRegexp(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+`))
			}

			if Config.GetIncludeTasks() {
				taskName := "get-env"

				Eventually(cf.Cf("run-task", appName, "--command", "env", "--name", taskName)).Should(Exit(0))

				Eventually(func() string {
					return getTaskState(appName)
				}).Should(Equal("SUCCEEDED"))

				var taskStdout string
				Eventually(func() string {
					appLogsSession := logs.Recent(appName)
					appLogsSession.Wait()

					taskStdout = string(appLogsSession.Out.Contents())

					return taskStdout
				}).Should(MatchRegexp("TASK.*VCAP_SERVICES=.*"))

				Expect(taskStdout).To(MatchRegexp("TASK.*LANG=en_US\\.UTF-8"))
				Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_INTERNAL_IP=.*"))
				Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_IP=.*"))
				Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_PORTS=.*"))
				Expect(taskStdout).To(MatchRegexp("TASK.*VCAP_APPLICATION=.*"))
				Expect(taskStdout).To(MatchRegexp("TASK.*VCAP_SERVICES=.*"))

				// these vars are set to the empty string (use m flag to make $ match eol)
				Expect(taskStdout).To(MatchRegexp("(?m)TASK.*CF_INSTANCE_ADDR=$"))
				Expect(taskStdout).To(MatchRegexp("(?m)TASK.*CF_INSTANCE_PORT=$"))
			}
		})
	})
})

func assertJsonParseable(env map[string]string, varNames ...string) {
	for _, varName := range varNames {
		value, ok := env[varName]
		Expect(ok).To(BeTrue())
		var vcapStruct map[string]interface{}
		err := json.Unmarshal([]byte(value), &vcapStruct)
		Expect(err).To(BeNil())
	}
}

func assertPresent(env map[string]string, varNames ...string) {
	for _, varName := range varNames {
		Expect(env).To(HaveKey(varName))
	}
}

func assertNotPresent(env map[string]string, varNames ...string) {
	for _, varName := range varNames {
		Expect(env).NotTo(HaveKey(varName))
	}
}

func getTaskState(appName string) string {
	listCommand := cf.Cf("tasks", appName).Wait()
	Expect(listCommand).To(Exit(0))
	listOutput := string(listCommand.Out.Contents())
	lines := strings.Split(listOutput, "\n")
	return strings.Fields(lines[3])[2]
}
