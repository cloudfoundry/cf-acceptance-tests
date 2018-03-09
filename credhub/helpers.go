package credhub

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func pushBroker() (chBrokerAppName, chServiceName, instanceName string) {
	chBrokerAppName = random_name.CATSRandomName("BRKR-CH")

	Expect(cf.Cf(
		"push", chBrokerAppName,
		"-b", Config.GetGoBuildpackName(),
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", assets.NewAssets().CredHubServiceBroker,
		"-f", assets.NewAssets().CredHubServiceBroker+"/manifest.yml",
		"-d", Config.GetAppsDomain(),
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing credhub-enabled service broker")

	existingEnvVar := string(cf.Cf("running-environment-variable-group").Wait(Config.DefaultTimeoutDuration()).Out.Contents())

	if !strings.Contains(existingEnvVar, "CREDHUB_API") {
		Expect(cf.Cf(
			"set-env", chBrokerAppName,
			"CREDHUB_API", Config.GetCredHubLocation(),
		).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting CREDHUB_API env var on credhub-enabled service broker")
	}

	chServiceName = random_name.CATSRandomName("SERVICE-NAME")
	Expect(cf.Cf(
		"set-env", chBrokerAppName,
		"SERVICE_NAME", chServiceName,
	).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting SERVICE_NAME env var on credhub-enabled service broker")

	Expect(cf.Cf(
		"set-env", chBrokerAppName,
		"CREDHUB_CLIENT", Config.GetCredHubBrokerClientCredential(),
	).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting CREDHUB_CLIENT env var on credhub-enabled service broker")

	Expect(cf.Cf(
		"set-env", chBrokerAppName,
		"CREDHUB_SECRET", Config.GetCredHubBrokerClientSecret(),
	).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting CREDHUB_SECRET env var on credhub-enabled service broker")

	Expect(cf.Cf(
		"restart", chBrokerAppName,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed restarting credhub-enabled service broker")

	serviceUrl := "https://" + chBrokerAppName + "." + Config.GetAppsDomain()
	createServiceBroker := cf.Cf("create-service-broker", chBrokerAppName, "user", "pass", serviceUrl, "--space-scoped").Wait(Config.DefaultTimeoutDuration())
	Expect(createServiceBroker).To(Exit(0), "failed creating credhub-enabled service broker")

	instanceName = random_name.CATSRandomName("SVIN-CH")
	createService := cf.Cf("create-service", chServiceName, "credhub-read-plan", instanceName).Wait(Config.DefaultTimeoutDuration())
	Expect(createService).To(Exit(0), "failed creating credhub enabled service")
	return
}

func bindServiceAndStartApp(chServiceName, instanceName, appName string) *Session {
	app_helpers.SetBackend(appName)

	Expect(chServiceName).ToNot(Equal(""))
	setServiceName := cf.Cf("set-env", appName, "SERVICE_NAME", chServiceName).Wait(Config.DefaultTimeoutDuration())
	Expect(setServiceName).To(Exit(0), "failed setting SERVICE_NAME env var on app")

	existingEnvVar := string(cf.Cf("running-environment-variable-group").Wait(Config.DefaultTimeoutDuration()).Out.Contents())

	if !strings.Contains(existingEnvVar, "CREDHUB_API") {
		Expect(cf.Cf(
			"set-env", appName,
			"CREDHUB_API", Config.GetCredHubLocation(),
		).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting CREDHUB_API env var on app")
	}

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		TestSetup.RegularUserContext().TargetSpace()

		bindService := cf.Cf("bind-service", appName, instanceName).Wait(Config.DefaultTimeoutDuration())
		Expect(bindService).To(Exit(0), "failed binding app to service")
	})
	appStartSession := cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
	Expect(appStartSession).To(Exit(0))

	return appStartSession
}

func pushBuildpackApp() (string, string) {
	buildpackName := random_name.CATSRandomName("BPK")

	var err error
	tmpdir, err := ioutil.TempDir("", "buildpack_env")
	Expect(err).ToNot(HaveOccurred())
	defer os.RemoveAll(tmpdir)
	appPath, err := ioutil.TempDir(tmpdir, "matching-app")
	Expect(err).ToNot(HaveOccurred())

	buildpackPath, err := ioutil.TempDir(tmpdir, "matching-buildpack")
	Expect(err).ToNot(HaveOccurred())

	buildpackArchivePath := path.Join(buildpackPath, "buildpack.zip")

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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; } | nc -l \$PORT; done
EOF
`,
		},
	})
	_, err = os.Create(path.Join(appPath, "some-file"))
	Expect(err).ToNot(HaveOccurred())

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		createBuildpack := cf.Cf("create-buildpack", buildpackName, buildpackArchivePath, "100").Wait(Config.DefaultTimeoutDuration())
		Expect(createBuildpack).Should(Exit(0))
		Expect(createBuildpack).Should(Say("Creating"))
		Expect(createBuildpack).Should(Say("OK"))
		Expect(createBuildpack).Should(Say("Uploading"))
		Expect(createBuildpack).Should(Say("OK"))
	})

	appName := random_name.CATSRandomName("APP")
	Expect(cf.Cf("push", appName,
		"--no-start",
		"-b", buildpackName,
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", appPath,
		"-d", Config.GetAppsDomain(),
	).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

	return appName, buildpackName
}
