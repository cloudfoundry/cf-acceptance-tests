package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/vito/runtime-integration/helpers"
)

const simpleBuildpackPath = "../assets/buildpacks/simple-buildpack.zip"
const anotherBuildpackPath = "../assets/buildpacks/another-buildpack.zip"

const blankAppPath = "../assets/blank-app"
const superBlankAppPath = "../assets/super-blank-app"

var _ = Describe("An application using an admin buildpack", func() {
	var BuildpackName string

	BeforeEach(func() {
		BuildpackName = RandomName()
		AppName = RandomName()

		createBuildpack := Cf("create-buildpack", BuildpackName, simpleBuildpackPath, "0")
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
			push := Cf("push", AppName, "-p", blankAppPath)
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("Started"))
		})
	})

	Context("when the buildpack fails to detect", func() {
		It("fails to stage", func() {
			Expect(Cf("push", AppName, "-p", superBlankAppPath)).To(Say("Staging error"))
		})
	})

	Context("when the buildpack is deleted", func() {
		BeforeEach(func() {
			Expect(Cf("delete-buildpack", BuildpackName, "-f")).To(Say("OK"))
		})

		It("fails to stage", func() {
			Expect(Cf("push", AppName, "-p", blankAppPath)).To(Say("Staging error"))
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
			Expect(Cf("push", AppName, "-p", blankAppPath)).To(Say("Staging error"))
		})
	})
})
