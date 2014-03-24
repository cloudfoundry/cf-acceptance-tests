package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	"github.com/cloudfoundry/cf-acceptance-tests/apps/helpers"
	catsHelpers "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/zip"
)

var _ = Describe("An application using an admin buildpack", func() {
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

	BeforeEach(func() {
		AsUser(catsHelpers.AdminUserContext, func() {
			BuildpackName = RandomName()
			appName = RandomName()

			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

			err = helpers.GenerateBuildpack(buildpackPath, matchingFilename(appName))
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

			buildpathFile, err := os.Create(buildpackArchivePath)
			Expect(err).ToNot(HaveOccurred())

			err = Zip(filepath.Join(buildpackPath, "bin"), buildpathFile)
			Expect(err).ToNot(HaveOccurred())

			createBuildpack := Cf("create-buildpack", BuildpackName, buildpackArchivePath, "0")
			Expect(createBuildpack).To(Say("Creating"))
			Expect(createBuildpack).To(Say("OK"))
			Expect(createBuildpack).To(Say("Uploading"))
			Expect(createBuildpack).To(Say("OK"))
		})
	})

	AfterEach(func() {
		AsUser(catsHelpers.AdminUserContext, func() {
			Expect(Cf("delete-buildpack", BuildpackName, "-f")).To(Say("OK"))
		})
	})

	Context("when the buildpack is detected", func() {
		It("is used for the app", func() {
			push := Cf("push", appName, "-p", appPath)
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("App started"))
		})
	})

	Context("when the buildpack fails to detect", func() {
		BeforeEach(func() {
			err := os.Remove(path.Join(appPath, matchingFilename(appName)))
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails to stage", func() {
			Expect(Cf("push", appName, "-p", appPath)).To(Say("Staging error"))
		})
	})

	Context("when the buildpack is deleted", func() {
		BeforeEach(func() {
			AsUser(catsHelpers.AdminUserContext, func() {
				Expect(Cf("delete-buildpack", BuildpackName, "-f")).To(Say("OK"))
			})
		})

		It("fails to stage", func() {
			Expect(Cf("push", appName, "-p", appPath)).To(Say("Staging error"))
		})
	})

	PContext("when the buildpack is disabled", func() {
		BeforeEach(func() {
			var response QueryResponse

			ApiRequest("GET", "/v2/buildpacks?q=name:"+BuildpackName, &response)

			Expect(response.Resources).To(HaveLen(1))

			buildpackGuid := response.Resources[0].Metadata.Guid

			ApiRequest(
				"PUT",
				"/v2/buildpacks/"+buildpackGuid,
				nil,
				`{"enabled":false}`,
			)
		})

		It("fails to stage", func() {
			Expect(Cf("push", appName, "-p", appPath)).To(Say("Staging error"))
		})
	})
})
