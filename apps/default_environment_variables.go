package apps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = AppsDescribe("Default Environment Variables", func() {
	var createBuildpack = func() string {
		tmpPath, err := ioutil.TempDir("", "default-env-var-test")
		Expect(err).ToNot(HaveOccurred())

		buildpackArchivePath := path.Join(tmpPath, "buildpack.zip")

		archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
			{
				Name: "bin/compile",
				Body: fmt.Sprintf(`#!/usr/bin/env bash
env
# wait for the log lines to make it through
sleep 5
exit 1
`),
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
			if Config.GetBackend() != "diego" {
				Skip(skip_messages.SkipDiegoMessage)
			}
			appName = random_name.CATSRandomName("APP")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				if buildpackName != "" {
					Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				}
			})

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("applies default environment variables while staging apps", func() {
			buildpackName = random_name.CATSRandomName("BPK")
			buildpackZip := createBuildpack()

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			})

			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().HelloWorld,
				"--no-start",
				"-b", buildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
				"-d", Config.GetAppsDomain()),
				Config.DefaultTimeoutDuration(),
			).Should(Exit(0))

			app_helpers.SetBackend(appName)
			startSession := cf.Cf("start", appName)
			Eventually(startSession, Config.CfPushTimeoutDuration()).Should(Exit(1))

			stdout := string(startSession.Out.Contents())
			Expect(stdout).To(MatchRegexp("LANG=en_US\\.UTF-8"))
			Expect(stdout).To(MatchRegexp("CF_INSTANCE_ADDR=.*"))
			Expect(stdout).To(MatchRegexp("CF_INSTANCE_INTERNAL_IP=.*"))
			Expect(stdout).To(MatchRegexp("CF_INSTANCE_IP=.*"))
			Expect(stdout).To(MatchRegexp("CF_INSTANCE_PORT=.*"))
			Expect(stdout).To(MatchRegexp("CF_INSTANCE_PORTS=.*"))
			Expect(stdout).To(MatchRegexp("CF_STACK=.*"))
			Expect(stdout).To(MatchRegexp("VCAP_APPLICATION=.*"))
			Expect(stdout).To(MatchRegexp("VCAP_SERVICES=.*"))
		})

		It("applies default environment variables while running apps and tasks", func() {
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"--no-start",
				"-b", "binary_buildpack",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-d", Config.GetAppsDomain()),
				Config.DefaultTimeoutDuration(),
			).Should(Exit(0))

			app_helpers.SetBackend(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))

			envLines := helpers.CurlApp(Config, appName, "/env")

			var env map[string]string
			err := json.Unmarshal([]byte(envLines), &env)
			Expect(err).To(BeNil())

			Expect(env["LANG"]).To(Equal("en_US.UTF-8"))
			assertJsonParseable(env, "VCAP_APPLICATION", "VCAP_SERVICES")
			assertPresent(env,
				"CF_INSTANCE_ADDR",
				"CF_INSTANCE_GUID",
				"CF_INSTANCE_INDEX",
				"CF_INSTANCE_INTERNAL_IP",
				"CF_INSTANCE_IP",
				"CF_INSTANCE_PORT",
				"CF_INSTANCE_PORTS",
				"VCAP_APP_HOST",
				"VCAP_APP_PORT",
			)

			taskName := "get-env"

			Eventually(cf.Cf(
				"run-task", appName, "env", "--name", taskName),
				Config.DefaultTimeoutDuration(),
			).Should(Exit(0))

			Eventually(func() string {
				return getTaskDetails(appName)[2]
			}, Config.DefaultTimeoutDuration()).Should(Equal("SUCCEEDED"))

			var taskStdout string
			Eventually(func() string {
				appLogsSession := logs.Tail(Config.GetUseLogCache(), appName)
				appLogsSession.Wait(Config.DefaultTimeoutDuration())

				taskStdout = string(appLogsSession.Out.Contents())

				return taskStdout
			}, Config.DefaultTimeoutDuration()).Should(MatchRegexp("TASK.*VCAP_SERVICES=.*"))

			Expect(taskStdout).To(MatchRegexp("TASK.*LANG=en_US\\.UTF-8"))
			Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_ADDR=.*"))
			Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_INTERNAL_IP=.*"))
			Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_IP=.*"))
			Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_PORT=.*"))
			Expect(taskStdout).To(MatchRegexp("TASK.*CF_INSTANCE_PORTS=.*"))
			Expect(taskStdout).To(MatchRegexp("TASK.*VCAP_APPLICATION=.*"))
			Expect(taskStdout).To(MatchRegexp("TASK.*VCAP_SERVICES=.*"))
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
		_, ok := env[varName]
		Expect(ok).To(BeTrue())
	}
}

func getTaskDetails(appName string) []string {
	listCommand := cf.Cf("tasks", appName).Wait(Config.DefaultTimeoutDuration())
	Expect(listCommand).To(Exit(0))
	listOutput := string(listCommand.Out.Contents())
	lines := strings.Split(listOutput, "\n")
	return strings.Fields(lines[4])
}
