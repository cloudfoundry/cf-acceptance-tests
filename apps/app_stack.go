package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Specifying a specific Stack", func() {
	var (
		appName       string
		BuildpackName string

		appPath string

		buildpackPath        string
		buildpackArchivePath string

		tmpdir string
	)

	matchingFilename := func(appName string) string {
		return fmt.Sprintf("stack-match-%s", appName)
	}

	BeforeEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			BuildpackName = CATSRandomName("BPK")
			appName = CATSRandomName("APP")

			var err error
			tmpdir, err = ioutil.TempDir("", "stack")
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

sleep 5

cat /etc/lsb-release

sleep 10
`,
				},
				{
					Name: "bin/detect",
					Body: fmt.Sprintf(`#!/bin/bash

if [ -f "${1}/%s" ]; then
  echo Simple
else
  echo no
  exit 1
fi
`, matchingFilename(appName)),
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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo -e "\$(cat /etc/lsb-release)"; } | nc -l \$PORT; done
EOF
`,
				},
			})
			_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := cf.Cf("create-buildpack", BuildpackName, buildpackArchivePath, "0").Wait(Config.DefaultTimeoutDuration())
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("delete-buildpack", BuildpackName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})

		os.RemoveAll(tmpdir)
	})

	It("uses cflinuxfs2 for staging and running", func() {
		stackName := "cflinuxfs2"
		expected_lsb_release := "DISTRIB_CODENAME=trusty"

		Expect(cf.Cf("push", appName,
			"--no-start",
			"-b", BuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", appPath,
			"-s", stackName,
			"-d", Config.GetAppsDomain(),
		).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
		Expect(start).To(Exit(0))
		Expect(start).To(Say(expected_lsb_release))
		Expect(start).To(Say(""))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expected_lsb_release))
	})
})
