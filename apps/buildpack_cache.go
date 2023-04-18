package apps

import (
	"fmt"
	"os"
	"path"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Buildpack cache", func() {
	var (
		appName       string
		BuildpackName string

		appPath string

		buildpackPath        string
		buildpackArchivePath string
	)

	SkipOnK8s("Custom buildpacks not yet supported")

	matchingFilename := func(appName string) string {
		return fmt.Sprintf("buildpack-for-buildpack-cache-test-%s", appName)
	}

	BeforeEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			BuildpackName = CATSRandomName("BPK")
			appName = CATSRandomName("APP")

			tmpdir, err := os.MkdirTemp(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir

			tmpdir, err = os.MkdirTemp(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

			archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
				{
					Name: "bin/compile",
					Body: `#!/usr/bin/env bash

mkdir -p $1 $2
if [ -f "$2/cached-file" ]; then
	cp $2/cached-file $1/content
else
	echo "cache not found" > $1/content
fi

echo "here's a cache" > $2/cached-file
`,
				},
				{
					Name: "bin/detect",
					Body: fmt.Sprintf(`#!/bin/bash

if [ -f "${1}/%s" ]; then
  echo Buildpack that needs cache
else
  echo no
  exit 1
fi
`, matchingFilename(appName)),
				},
				{
					Name: "bin/release",
					Body: `#!/usr/bin/env bash

content=$(cat $1/content)

cat <<EOF
---
config_vars:
  PATH: bin:/usr/local/bin:/usr/bin:/bin
  FROM_BUILD_PACK: "yes"
default_process_types:
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "custom buildpack contents - $content"; } | nc -q 1 -l \$PORT; done
EOF
`,
				},
			})

			_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := cf.Cf("create-buildpack", BuildpackName, buildpackArchivePath, "1").Wait()
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
			Expect(cf.Cf("delete-buildpack", BuildpackName, "-f").Wait()).To(Exit(0))
		})
	})

	It("uses the buildpack cache after first staging", func() {
		Expect(cf.Cf("push", appName,
			"-t", "120",
			"-b", BuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", appPath,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		appGuid := app_helpers.GetAppGuid(appName)

		processes := GetProcesses(appGuid, appName)
		// start command is redacted on list fetch
		process := GetProcessByGuid(processes[0].Guid)
		Expect(process.Command).Should(ContainSubstring("custom buildpack contents - cache not found"))

		time.Sleep(Config.SleepTimeoutDuration())

		restage := cf.Cf("restage", appName).Wait(Config.CfPushTimeoutDuration())
		Expect(restage).To(Exit(0))

		processes = GetProcesses(appGuid, appName)
		process = GetProcessByGuid(processes[0].Guid)

		Expect(process.Command).Should(ContainSubstring("custom buildpack contents - here's a cache"))

	})
})
