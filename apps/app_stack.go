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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Specifying a specific stack", func() {
	var (
		appName       string
		buildpackName string

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
			buildpackName = CATSRandomName("BPK")
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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo -e "\$(cat /etc/lsb-release)"; } | nc -q 1 -l \$PORT; done
EOF
`,
				},
			})
			_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := cf.Cf("create-buildpack", buildpackName, buildpackArchivePath, "1").Wait()
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait()).To(Exit(0))
		})

		os.RemoveAll(tmpdir)
	})

	Context("when stack(s) are specified", func() {
		It("uses stack(s) for staging and running", func() {
			stacks := Config.GetStacks()
			if len(stacks) == 0 {
				Skip(skip_messages.SkipNoAlternateStacksMessage)
			}

			for _, stackName := range stacks {
				By(fmt.Sprintf("testing stack: %s", stackName))

				var expectedLSBRelease string
				switch stackName {
				case "cflinuxfs3":
					expectedLSBRelease = "DISTRIB_CODENAME=bionic"
				}

				push := cf.Cf("push", appName,
					"-b", buildpackName,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", appPath,
					"-s", stackName,
				).Wait(Config.CfPushTimeoutDuration())
				Expect(push).To(Exit(0))
				Expect(push).To(Say(expectedLSBRelease))

				Eventually(func() string {
					return helpers.CurlAppRoot(Config, appName)
				}).Should(ContainSubstring(expectedLSBRelease))
			}
		})
	})
})
