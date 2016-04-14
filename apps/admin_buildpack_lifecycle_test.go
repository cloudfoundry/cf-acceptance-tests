package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gbytes"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"
	archive_helpers "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/pivotal-golang/archiver/extractor/test_helper"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
)

var _ = Describe("Admin Buildpacks", func() {
	var (
		appName       string
		BuildpackName string

		appPath string

		buildpackPath        string
		buildpackArchivePath string
	)

	matchingFilename := func(appName string) string {
		return fmt.Sprintf("simple-buildpack-please-match-%s", appName)
	}

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
	})

	type appConfig struct {
		Empty bool
	}

	setupBadDetectBuildpack := func(appConfig appConfig) {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			BuildpackName = RandomName()
			appName = PrefixedRandomName("CATS-APP-")

			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

			archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
				{
					Name: "bin/compile",
					Body: `#!/usr/bin/env bash

sleep 5 # give loggregator time to start streaming the logs

echo "Staging with Simple Buildpack"

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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; } | nc -l \$PORT; done
EOF
`,
				},
			})

			if !appConfig.Empty {
				_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
				Expect(err).ToNot(HaveOccurred())
			}

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := cf.Cf("create-buildpack", BuildpackName, buildpackArchivePath, "100").Wait(DEFAULT_TIMEOUT)
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	setupBadCompileBuildpack := func(appConfig appConfig) {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			BuildpackName = RandomName()
			appName = PrefixedRandomName("CATS-APP-")

			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

			archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
				{
					Name: "bin/compile",
					Body: `#!/usr/bin/env bash

exit 1
`,
				},
				{
					Name: "bin/detect",
					Body: fmt.Sprintf(`#!/bin/bash

echo Simple
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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; } | nc -l \$PORT; done
EOF
`,
				},
			})

			if !appConfig.Empty {
				_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
				Expect(err).ToNot(HaveOccurred())
			}

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := cf.Cf("create-buildpack", BuildpackName, buildpackArchivePath, "0").Wait(DEFAULT_TIMEOUT)
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	setupBadReleaseBuildpack := func(appConfig appConfig) {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			BuildpackName = RandomName()
			appName = PrefixedRandomName("CATS-APP-")

			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

			archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
				{
					Name: "bin/compile",
					Body: `#!/usr/bin/env bash

echo Pass compile
`,
				},
				{
					Name: "bin/detect",
					Body: fmt.Sprintf(`#!/bin/bash

echo Pass Detect
`, matchingFilename(appName)),
				},
				{
					Name: "bin/release",
					Body: `#!/usr/bin/env bash

exit 1
`,
				},
			})

			if !appConfig.Empty {
				_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
				Expect(err).ToNot(HaveOccurred())
			}

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := cf.Cf("create-buildpack", BuildpackName, buildpackArchivePath, "0").Wait(DEFAULT_TIMEOUT)
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	deleteBuildpack := func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("delete-buildpack", BuildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	}

	deleteApp := func() {
		command := cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)
		Expect(command).To(Exit(0))
		Expect(command).To(Say(fmt.Sprintf("Deleting app %s", appName)))
	}

	itIsUsedForTheApp := func() {
		Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
		Expect(start).To(Exit(0))
		Expect(start).To(Say("Staging with Simple Buildpack"))
	}

	itDoesNotDetectForEmptyApp := func() {
		Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
		Expect(start).To(Exit(1))
		Expect(start).To(Say("NoAppDetectedError"))
	}

	itDoesNotDetectWhenBuildpackDisabled := func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			var response cf.QueryResponse

			cf.ApiRequest("GET", "/v2/buildpacks?q=name:"+BuildpackName, &response, DEFAULT_TIMEOUT)

			Expect(response.Resources).To(HaveLen(1))

			buildpackGuid := response.Resources[0].Metadata.Guid

			cf.ApiRequest(
				"PUT",
				"/v2/buildpacks/"+buildpackGuid,
				nil,
				DEFAULT_TIMEOUT,
				`{"enabled":false}`,
			)
		})

		Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
		Expect(start).To(Exit(1))
		Expect(start).To(Say("NoAppDetectedError"))
	}

	itDoesNotDetectWhenBuildpackDeleted := func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("delete-buildpack", BuildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
		Expect(start).To(Exit(1))
		Expect(start).To(Say("NoAppDetectedError"))
	}

	itRaisesBuildpackCompileFailedError := func() {
		Expect(cf.Cf("push", appName, "--no-start", "-b", BuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
		Expect(start).To(Exit(1))
		Expect(start).To(Say("BuildpackCompileFailed"))
	}

	itRaisesBuildpackReleaseFailedError := func() {
		Expect(cf.Cf("push", appName, "--no-start", "-b", BuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.SetBackend(appName)

		start := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
		Expect(start).To(Exit(1))
		Expect(start).To(Say("BuildpackReleaseFailed"))
	}

	Context("when the buildpack is not specified", func() {
		It("runs the app only if the buildpack is detected", func() {
			// Tests that rely on buildpack detection must be run in serial,
			// but ginkgo doesn't allow specific blocks to be marked as serial-only
			// so we manually mimic setup/teardown pattern here

			setupBadDetectBuildpack(appConfig{Empty: false})
			itIsUsedForTheApp()
			deleteApp()
			deleteBuildpack()

			setupBadDetectBuildpack(appConfig{Empty: true})
			itDoesNotDetectForEmptyApp()
			deleteApp()
			deleteBuildpack()

			setupBadDetectBuildpack(appConfig{Empty: false})
			itDoesNotDetectWhenBuildpackDisabled()
			deleteApp()
			deleteBuildpack()

			setupBadDetectBuildpack(appConfig{Empty: false})
			itDoesNotDetectWhenBuildpackDeleted()
			deleteApp()
			deleteBuildpack()
		})
	})

	Context("when the buildpack compile fails", func() {
		// This test used to be part of inigo and with the extraction of CC bridge, we want to ensure
		// that user facing errors are correctly propagated from a garden container out of the system.

		It(diegoUnsupportedTag+"the user receives a BuildpackCompileFailed error", func() {
			setupBadCompileBuildpack(appConfig{Empty: false})
			itRaisesBuildpackCompileFailedError()
			deleteApp()
			deleteBuildpack()
		})
	})

	Context("when the buildpack release fails", func() {
		// This test used to be part of inigo and with the extraction of CC bridge, we want to ensure
		// that user facing errors are correctly propagated from a garden container out of the system.

		It(diegoUnsupportedTag+"the user receives a BuildpackReleaseFailed error", func() {
			setupBadReleaseBuildpack(appConfig{Empty: false})
			itRaisesBuildpackReleaseFailedError()
			deleteApp()
			deleteBuildpack()
		})
	})
})
