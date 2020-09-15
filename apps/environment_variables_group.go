package apps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"time"

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
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
)

var _ = AppsDescribe("Environment Variables Groups", func() {
	type EnvVarResponse struct{
		Vars map[string]string `json:"var"`
	}

	var createBuildpack = func(envVarName string) string {
		tmpPath, err := ioutil.TempDir("", "env-group-staging")
		Expect(err).ToNot(HaveOccurred())

		buildpackArchivePath := path.Join(tmpPath, "buildpack.zip")

		archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
			{
				Name: "bin/compile",
				Body: fmt.Sprintf(`#!/usr/bin/env bash
sleep 5
echo $%s
exit 1
`, envVarName),
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

	var fetchEnvironmentVariables = func(groupType string) map[string]string {
		var session *Session
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			session = cf.Cf("curl", fmt.Sprintf("/v3/environment_variable_groups/%s", groupType)).Wait()
			Expect(session).To(Exit(0))
		})

		var envVarGroupResponse = EnvVarResponse{}
		err := json.Unmarshal(session.Out.Contents(), &envVarGroupResponse)
		Expect(err).NotTo(HaveOccurred())

		if envVarGroupResponse.Vars == nil {
			envVarGroupResponse.Vars = make(map[string]string)
		}
		return envVarGroupResponse.Vars
	}

	var marshalUpdatedEnv = func(envMap map[string]string) []byte {
		jsonObj, err := json.Marshal(envMap)
		Expect(err).NotTo(HaveOccurred())
		return jsonObj
	}

	var extendEnv = func(groupType, envVarName, envVarValue string) {
		envMap := fetchEnvironmentVariables(groupType)
		envMap[envVarName] = envVarValue
		jsonObj := marshalUpdatedEnv(envMap)

		command := fmt.Sprintf("set-%s-environment-variable-group", groupType)
		Expect(cf.Cf(command, string(jsonObj)).Wait()).To(Exit(0))
	}

	var revertExtendedEnv = func(groupType, envVarName string) {
		envMap := fetchEnvironmentVariables(groupType)
		jsonObj := marshalUpdatedEnv(envMap)

		apiUrl := fmt.Sprintf("/v3/environment_variable_groups/%s", groupType)
		Expect(cf.Cf("curl", apiUrl, "-X", "PATCH", "-d", string(jsonObj)).Wait()).To(Exit(0))
	}

	Context("Staging environment variable groups", func() {
		var appName string
		var buildpackName string
		var envVarName string

		SkipOnK8s("Custom buildpacks not yet supported")

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			envVarName = fmt.Sprintf("CATS_STAGING_TEST_VAR_%s", strconv.Itoa(int(time.Now().UnixNano())))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				revertExtendedEnv("staging", envVarName)
				if buildpackName != "" {
					Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait()).To(Exit(0))
				}
			})

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("Applies environment variables while staging apps", func() {
			buildpackName = random_name.CATSRandomName("BPK")
			buildpackZip := createBuildpack(envVarName)
			envVarValue := fmt.Sprintf("staging_env_value_%s", strconv.Itoa(int(time.Now().UnixNano())))

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				extendEnv("staging", envVarName, envVarValue)
				Expect(cf.Cf("create-buildpack", buildpackName, buildpackZip, "999").Wait()).To(Exit(0))
			})

			Expect(cf.Cf("push", appName, "-m", DEFAULT_MEMORY_LIMIT, "-b", buildpackName, "-p", assets.NewAssets().HelloWorld).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))
			Eventually(logs.Recent(appName)).Should(Say(envVarValue))
		})
	})

	Context("Running environment variable groups", func() {
		var appName string
		var envVarName string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			envVarName = fmt.Sprintf("CATS_RUNNING_TEST_VAR_%s", strconv.Itoa(int(time.Now().UnixNano())))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				revertExtendedEnv("running", envVarName)
			})

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("Applies correct environment variables while running apps", func() {
			envVarValue := fmt.Sprintf("running_env_value_%s", strconv.Itoa(int(time.Now().UnixNano())))
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				extendEnv("running", envVarName, envVarValue)
			})

			Expect(cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			env := helpers.CurlApp(Config, appName, "/env.json")
			Expect(env).To(ContainSubstring(envVarValue))
		})
	})
})
