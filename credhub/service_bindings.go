package credhub

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"encoding/json"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = CredhubDescribe("service bindings", func() {
	var (
		chBrokerAppName string
		chServiceName   string
		instanceName    string
		appStartSession *Session
	)

	BeforeEach(func() {
		TestSetup.RegularUserContext().TargetSpace()
		cf.Cf("target", "-o", TestSetup.RegularUserContext().Org)
		Expect(string(cf.Cf("running-environment-variable-group").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(ContainSubstring("CREDHUB_API"), "CredHub API environment not set")

		chBrokerAppName = random_name.CATSRandomName("BRKR-CH")

		Expect(cf.Cf(
			"push", chBrokerAppName,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().CredHubServiceBroker,
			"-f", assets.NewAssets().CredHubServiceBroker+"/manifest.yml",
			"-d", Config.GetAppsDomain(),
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing credhub-enabled service broker")

		chServiceName = random_name.CATSRandomName("SERVICE-NAME")
		Expect(cf.Cf(
			"set-env", chBrokerAppName,
			"SERVICE_NAME", chServiceName,
		).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting SERVICE_NAME env var on credhub-enabled service broker")

		Expect(cf.Cf(
			"restart", chBrokerAppName,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed restarting credhub-enabled service broker")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			serviceUrl := "https://" + chBrokerAppName + "." + Config.GetAppsDomain()
			createServiceBroker := cf.Cf("create-service-broker", chBrokerAppName, Config.GetAdminUser(), Config.GetAdminPassword(), serviceUrl).Wait(Config.DefaultTimeoutDuration())
			Expect(createServiceBroker).To(Exit(0), "failed creating credhub-enabled service broker")

			enableAccess := cf.Cf("enable-service-access", chServiceName, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
			Expect(enableAccess).To(Exit(0), "failed to enable service access for credhub-enabled broker")

			TestSetup.RegularUserContext().TargetSpace()
			instanceName = random_name.CATSRandomName("SVIN-CH")
			createService := cf.Cf("create-service", chServiceName, "credhub-read-plan", instanceName).Wait(Config.DefaultTimeoutDuration())
			Expect(createService).To(Exit(0), "failed creating credhub enabled service")
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			TestSetup.RegularUserContext().TargetSpace()

			Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("delete-service-broker", chBrokerAppName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})
	})

	bindServiceAndStartApp := func(appName string) {
		app_helpers.SetBackend(appName)

		Expect(chServiceName).ToNot(Equal(""))
		setServiceName := cf.Cf("set-env", appName, "SERVICE_NAME", chServiceName).Wait(Config.DefaultTimeoutDuration())
		Expect(setServiceName).To(Exit(0), "failed setting SERVICE_NAME env var on app")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			TestSetup.RegularUserContext().TargetSpace()

			bindService := cf.Cf("bind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
			Expect(bindService).To(Exit(0), "failed binding app to service")
		})
		appStartSession = cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
		Expect(appStartSession).To(Exit(0))
	}

	Context("during staging", func() {
		var (
			buildpackName string
			appName       string
			appPath       string

			buildpackPath        string
			buildpackArchivePath string

			tmpdir string
		)
		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				buildpackName = random_name.CATSRandomName("BPK")
				appName = random_name.CATSRandomName("APP")

				var err error
				tmpdir, err = ioutil.TempDir("", "buildpack_env")
				Expect(err).ToNot(HaveOccurred())
				appPath, err = ioutil.TempDir(tmpdir, "matching-app")
				Expect(err).ToNot(HaveOccurred())

				buildpackPath, err = ioutil.TempDir(tmpdir, "matching-buildpack")
				Expect(err).ToNot(HaveOccurred())

				buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

				archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
					{
						Name: "bin/compile",
						Body: `#!/usr/bin/env bash
echo COMPILING... really just dumping env...
env
`,
					},
					{
						Name: "bin/detect",
						Body: `#!/bin/bash

exit 0
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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; } | nc -l \$PORT; done
EOF
`,
					},
				})
				_, err = os.Create(path.Join(appPath, "some-file"))
				Expect(err).ToNot(HaveOccurred())

				createBuildpack := cf.Cf("create-buildpack", buildpackName, buildpackArchivePath, "0").Wait(Config.DefaultTimeoutDuration())
				Expect(createBuildpack).Should(Exit(0))
				Expect(createBuildpack).Should(Say("Creating"))
				Expect(createBuildpack).Should(Say("OK"))
				Expect(createBuildpack).Should(Say("Uploading"))
				Expect(createBuildpack).Should(Say("OK"))

			})
			Expect(cf.Cf("push", appName,
				"--no-start",
				"-b", buildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", appPath,
				"-d", Config.GetAppsDomain(),
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			app_helpers.SetBackend(appName)
			bindServiceAndStartApp(appName)
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			})

			os.RemoveAll(tmpdir)
		})

		NonAssistedCredhubDescribe("", func() {
			It("still contains CredHub references in VCAP_SERVICES", func() {
				Expect(appStartSession).NotTo(Say("pinkyPie"))
				Expect(appStartSession).NotTo(Say("rainbowDash"))
				Expect(appStartSession).To(Say("credhub-ref"))
			})
		})

		AssistedCredhubDescribe("", func() {
			It("has CredHub references in VCAP_SERVICES interpolated", func() {
				Expect(appStartSession).To(Say(`{"password":"rainbowDash","user-name":"pinkyPie"}`))
				Expect(appStartSession).NotTo(Say("credhub-ref"))
			})
		})
	})

	Context("during runtime", func() {
		Describe("service bindings to credhub enabled broker", func() {
			var appName, appURL string
			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP-CH")
				appURL = "https://" + appName + "." + Config.GetAppsDomain()
			})

			AfterEach(func() {
				app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
				app_helpers.AppReport(chBrokerAppName, Config.DefaultTimeoutDuration())

				workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
					TestSetup.RegularUserContext().TargetSpace()
					unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
					Expect(unbindService).To(Exit(0), "failed unbinding app and service")

					Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				})
			})

			NonAssistedCredhubDescribe("", func() {
				BeforeEach(func() {
					createApp := cf.Cf(
						"push", appName,
						"--no-start",
						"-b", Config.GetJavaBuildpackName(),
						"-m", "1024M",
						"-p", assets.NewAssets().CredHubEnabledApp,
						"-d", Config.GetAppsDomain(),
					).Wait(Config.CfPushTimeoutDuration())
					Expect(createApp).To(Exit(0), "failed creating credhub-enabled app")
					bindServiceAndStartApp(appName)
				})

				It("the broker returns credhub-ref in the credentials block", func() {
					appEnv := string(cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")
				})

				It("the bound app retrieves the credentials for the ref from CredHub", func() {
					curlCmd := helpers.CurlSkipSSL(true, appURL+"/test").Wait(Config.DefaultTimeoutDuration())
					Expect(curlCmd).To(Exit(0))

					bytes := curlCmd.Out.Contents()
					var response struct {
						UserName string `json:"user-name"`
						Password string `json:"password"`
					}

					json.Unmarshal(bytes, &response)
					Expect(response.UserName).To(Equal("pinkyPie"))
					Expect(response.Password).To(Equal("rainbowDash"))
				})
			})

			AssistedCredhubDescribe("", func() {
				BeforeEach(func() {
					createApp := cf.Cf(
						"push", appName,
						"--no-start",
						"-b", Config.GetBinaryBuildpackName(),
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Catnip,
						"-c", "./catnip",
						"-d", Config.GetAppsDomain(),
					).Wait(Config.CfPushTimeoutDuration())
					Expect(createApp).To(Exit(0), "failed creating credhub-enabled app")
					bindServiceAndStartApp(appName)
				})

				It("the broker returns credhub-ref in the credentials block", func() {
					appEnv := string(cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration()).Out.Contents())
					Expect(appEnv).To(ContainSubstring("credentials"), "credential block missing from service")
					Expect(appEnv).To(ContainSubstring("credhub-ref"), "credhub-ref not found")
				})

				It("the bound app gets CredHub refs in VCAP_SERVICES interpolated", func() {
					curlCmd := helpers.CurlSkipSSL(true, appURL+"/env/VCAP_SERVICES").Wait(Config.DefaultTimeoutDuration())
					Expect(curlCmd).To(Exit(0))

					bytes := curlCmd.Out.Contents()
					Expect(string(bytes)).To(ContainSubstring(`"rainbowDash"`))
					Expect(string(bytes)).To(ContainSubstring(`"pinkyPie"`))
				})
			})
		})
	})
})
