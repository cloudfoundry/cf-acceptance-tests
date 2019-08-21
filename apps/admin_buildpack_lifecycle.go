package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	archive_helpers "code.cloudfoundry.org/archiver/extractor/test_helper"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Admin Buildpacks", func() {
	var (
		appName        string
		appNames       []string
		buildpackName  string
		buildpackNames []string

		appPath string

		buildpackPath        string
		buildpackArchivePath string
	)

	noAppDetectedErrorRegexp := "NoAppDetectedError|An app was not successfully detected by any available buildpack"
	buildpackCompileFailedRegexp := "BuildpackCompileFailed|App staging failed in the buildpack compile phase"
	buildpackReleaseFailedRegexp := "BuildpackReleaseFailed|App staging failed in the buildpack release phase"

	matchingFilename := func(appName string) string {
		return fmt.Sprintf("simple-buildpack-please-match-%s", appName)
	}

	AfterEach(func() {
		app_helpers.AppReport(appName)
		for _, name := range buildpackNames {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("delete-buildpack", name, "-f").Wait()).To(Exit(0))
			})
		}
		for _, name := range appNames {
			command := cf.Cf("delete", name, "-f", "-r").Wait()
			Expect(command).To(Exit(0))
			Expect(command).To(Say(fmt.Sprintf("Deleting app %s", name)))
		}
		buildpackNames = []string{}
		appNames = []string{}
	})

	type appConfig struct {
		Empty bool
	}

	setupBadDetectBuildpack := func(appConfig appConfig) {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			buildpackName = CATSRandomName("BPK")
			buildpackNames = append(buildpackNames, buildpackName)
			appName = CATSRandomName("APP")
			appNames = append(appNames, appName)

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

exit 0
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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; } | nc -q 1 -l \$PORT; done
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

			createBuildpack := cf.Cf("create-buildpack", buildpackName, buildpackArchivePath, "1").Wait()
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	setupBadCompileBuildpack := func(appConfig appConfig) {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			buildpackName = CATSRandomName("BPK")
			buildpackNames = append(buildpackNames, buildpackName)
			appName = CATSRandomName("APP")
			appNames = append(appNames, appName)

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

			createBuildpack := cf.Cf("create-buildpack", buildpackName, buildpackArchivePath, "1").Wait()
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	setupBadReleaseBuildpack := func(appConfig appConfig) {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			buildpackName = CATSRandomName("BPK")
			buildpackNames = append(buildpackNames, buildpackName)
			appName = CATSRandomName("APP")
			appNames = append(appNames, appName)

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

if [ -f "${1}/%s" ]; then
  echo Pass Detect
else
  echo no
  exit 1
fi
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

			createBuildpack := cf.Cf("create-buildpack", buildpackName, buildpackArchivePath, "1").Wait()
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	itIsUsedForTheApp := func() {
		push := cf.Cf("push", appName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())

		Expect(push).To(Exit(0))
		appOutput := cf.Cf("app", appName).Wait()
		Expect(appOutput).To(Say("buildpacks?:\\s+Simple"))
	}

	itDoesNotDetectForEmptyApp := func() {
		push := cf.Cf("push", appName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())

		Expect(push).To(Exit(1))
		Expect(combineOutput(push.Out, push.Err)).To(Say(noAppDetectedErrorRegexp))
	}

	itDoesNotDetectWhenBuildpackDisabled := func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("update-buildpack", buildpackName, "--disable").Wait()).To(Exit(0))
		})

		push := cf.Cf("push", appName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
		Expect(push).To(Exit(1))
		Expect(combineOutput(push.Out, push.Err)).To(Say(noAppDetectedErrorRegexp))
	}

	itDoesNotDetectWhenBuildpackDeleted := func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("delete-buildpack", buildpackName, "-f").Wait()).To(Exit(0))
		})
		push := cf.Cf("push", appName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath).Wait(Config.CfPushTimeoutDuration())
		Expect(push).To(Exit(1))
		Expect(combineOutput(push.Out, push.Err)).To(Say(noAppDetectedErrorRegexp))
	}

	itRaisesBuildpackCompileFailedError := func() {
		push := cf.Cf("push",
			appName,
			"-b", buildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", appPath,
		).Wait(Config.CfPushTimeoutDuration())
		Expect(push).To(Exit(1))
		Expect(combineOutput(push.Out, push.Err)).To(Say(buildpackCompileFailedRegexp))
	}

	itRaisesBuildpackReleaseFailedError := func() {
		push := cf.Cf("push", appName, "-b", buildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
		Expect(push).To(Exit(1))
		Expect(combineOutput(push.Out, push.Err)).To(Say(buildpackReleaseFailedRegexp))
	}

	Context("when the buildpack is not specified", func() {
		It("runs the app only if the buildpack is detected", func() {
			// Tests that rely on buildpack detection must be run in serial,
			// but ginkgo doesn't allow specific blocks to be marked as serial-only
			// so we manually mimic setup/teardown pattern here

			setupBadDetectBuildpack(appConfig{Empty: false})
			itIsUsedForTheApp()

			setupBadDetectBuildpack(appConfig{Empty: true})
			itDoesNotDetectForEmptyApp()

			setupBadDetectBuildpack(appConfig{Empty: false})
			itDoesNotDetectWhenBuildpackDisabled()

			setupBadDetectBuildpack(appConfig{Empty: false})
			itDoesNotDetectWhenBuildpackDeleted()
		})
	})

	Context("when the buildpack is specified", func() {
		It("stages the app using the specified buildpack", func() {
			setupBadDetectBuildpack(appConfig{Empty: false})

			Expect(cf.Cf("push", appName, "-b", buildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			appOutput := cf.Cf("app", appName).Wait()
			Expect(appOutput).To(Say("buildpacks?:\\s+" + buildpackName))
		})

		It("fails if the specified buildpack is disabled", func() {
			setupBadDetectBuildpack(appConfig{Empty: false})

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("update-buildpack", buildpackName, "--disable").Wait()).To(Exit(0))
			})

			Expect(cf.Cf("push", appName, "-b", buildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", appPath, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))
		})
	})

	Context("when the buildpack compile fails", func() {
		// This test used to be part of inigo and with the extraction of CC bridge, we want to ensure
		// that user facing errors are correctly propagated from a garden container out of the system.

		It("the user receives a BuildpackCompileFailed error", func() {
			setupBadCompileBuildpack(appConfig{Empty: false})
			itRaisesBuildpackCompileFailedError()
		})
	})

	Context("when the buildpack release fails", func() {
		// This test used to be part of inigo and with the extraction of CC bridge, we want to ensure
		// that user facing errors are correctly propagated from a garden container out of the system.

		It("the user receives a BuildpackReleaseFailed error", func() {
			setupBadReleaseBuildpack(appConfig{Empty: false})
			itRaisesBuildpackReleaseFailedError()
		})
	})
})

func combineOutput(outBuffer *Buffer, errBuffer *Buffer) *Buffer {
	combinedOutput := BufferWithBytes(outBuffer.Contents())
	combinedOutput.Write(errBuffer.Contents())
	return combinedOutput
}
