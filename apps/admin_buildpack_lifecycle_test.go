package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	"github.com/pivotal-cf-experimental/cf-acceptance-tests/buildpack_generator"
	. "github.com/pivotal-cf-experimental/cf-acceptance-tests/helpers"
)

var _ = Describe("An application using an admin buildpack", func() {
	var (
		AppName       string
		BuildpackName string

		appPath string

		buildpackPath        string
		buildpackArchivePath string
	)

	matchingFilename := func(appName string) string {
		return fmt.Sprintf("simple-buildpack-please-match-%s", appName)
	}

	BeforeEach(func() {
		BuildpackName = RandomName()
		AppName = RandomName()

		tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
		Expect(err).ToNot(HaveOccured())

		appPath = tmpdir

		tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
		Expect(err).ToNot(HaveOccured())

		buildpackPath = tmpdir
		buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

		err = buildpack_generator.GenerateBuildpack(buildpackPath, matchingFilename(AppName))
		Expect(err).ToNot(HaveOccured())

		_, err = os.Create(path.Join(appPath, matchingFilename(AppName)))
		Expect(err).ToNot(HaveOccured())

		_, err = os.Create(path.Join(appPath, "some-file"))
		Expect(err).ToNot(HaveOccured())

		zipBuildpack := Run("bash", "-c", fmt.Sprintf("cd %s && zip -r %s bin", buildpackPath, buildpackArchivePath))
		Expect(zipBuildpack).To(ExitWith(0))

		createBuildpack := Cf("create-buildpack", BuildpackName, buildpackArchivePath, "0")
		Expect(createBuildpack).To(Say("Creating"))
		Expect(createBuildpack).To(Say("OK"))
		Expect(createBuildpack).To(Say("Uploading"))
		Expect(createBuildpack).To(Say("OK"))
	})

	AfterEach(func() {
		Expect(Cf("delete-buildpack", BuildpackName, "-f")).To(Say("OK"))
	})

	Context("when the buildpack is detected", func() {
		It("is used for the app", func() {
			push := Cf("push", AppName, "-p", appPath)
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("App started"))
		})
	})

	Context("when the buildpack fails to detect", func() {
		BeforeEach(func() {
			err := os.Remove(path.Join(appPath, matchingFilename(AppName)))
			Expect(err).ToNot(HaveOccured())
		})

		It("fails to stage", func() {
			Expect(Cf("push", AppName, "-p", appPath)).To(Say("Staging error"))
		})
	})

	Context("when the buildpack is deleted", func() {
		BeforeEach(func() {
			Expect(Cf("delete-buildpack", BuildpackName, "-f")).To(Say("OK"))
		})

		It("fails to stage", func() {
			Expect(Cf("push", AppName, "-p", appPath)).To(Say("Staging error"))
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
			Expect(Cf("push", AppName, "-p", appPath)).To(Say("Staging error"))
		})
	})
})
