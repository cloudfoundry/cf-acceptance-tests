package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
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

	type AppConfig struct {
		Empty bool
	}

	appWithContent := func() AppConfig {
		return AppConfig{
			Empty: false,
		}
	}

	emptyApp := func() AppConfig {
		return AppConfig{
			Empty: true,
		}
	}

	setupBuildpack := func(appConfig AppConfig) {
		AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
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

			createBuildpack := Cf("create-buildpack", BuildpackName, buildpackArchivePath, "0").Wait(DEFAULT_TIMEOUT)
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))
		})
	}

	deleteBuildpack := func() {
		AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(Cf("delete-buildpack", BuildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	}

	itIsUsedForTheApp := func() {
		push := Cf("push", appName, "-m", "128M", "-p", appPath).Wait(CF_PUSH_TIMEOUT)
		Expect(push).To(Exit(0))
		Expect(push).To(Say("Staging with Simple Buildpack"))
	}

	itDoesNotDetectForEmptyApp := func() {
		push := Cf("push", appName, "-m", "128M", "-p", appPath, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)
		Expect(push).To(Exit(1))
		Expect(push).To(Say("NoAppDetectedError"))
	}

	itDoesNotDetectWhenBuildpackDisabled := func() {
		AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			var response QueryResponse

			ApiRequest("GET", "/v2/buildpacks?q=name:"+BuildpackName, &response, DEFAULT_TIMEOUT)

			Expect(response.Resources).To(HaveLen(1))

			buildpackGuid := response.Resources[0].Metadata.Guid

			ApiRequest(
				"PUT",
				"/v2/buildpacks/"+buildpackGuid,
				nil,
				DEFAULT_TIMEOUT,
				`{"enabled":false}`,
			)
		})

		push := Cf("push", appName, "-m", "128M", "-p", appPath, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)
		Expect(push).To(Exit(1))
		Expect(push).To(Say("NoAppDetectedError"))
	}

	itDoesNotDetectWhenBuildpackDeleted := func() {
		AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(Cf("delete-buildpack", BuildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
		push := Cf("push", appName, "-m", "128M", "-p", appPath).Wait(CF_PUSH_TIMEOUT)
		Expect(push).To(Exit(1))
		Expect(push).To(Say("NoAppDetectedError"))
	}

	Context("when the buildpack is not specified", func() {
		It("runs the app only if the buildpack is detected", func() {
			// Tests that rely on buildpack detection must be run in serial,
			// but ginkgo doesn't allow specific blocks to be marked as serial-only
			// so we manually mimic setup/teardown pattern here

			setupBuildpack(appWithContent())
			itIsUsedForTheApp()
			deleteBuildpack()

			setupBuildpack(emptyApp())
			itDoesNotDetectForEmptyApp()
			deleteBuildpack()

			setupBuildpack(appWithContent())
			itDoesNotDetectWhenBuildpackDisabled()
			deleteBuildpack()

			setupBuildpack(appWithContent())
			itDoesNotDetectWhenBuildpackDeleted()
			deleteBuildpack()
		})
	})
})
