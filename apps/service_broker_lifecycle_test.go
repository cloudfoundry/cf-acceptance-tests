package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

type Service struct {
	Meta		metadata
	Entity 	entity
}

type Entity struct {
	Label							string
	Provider					string
	Url								string
	Description				string
	LongDescription 	string
	Version						string
	InfoUrl						string
	Active						bool
	Bindable					bool
	UniqueId					int
	Extra							extra
	requires					*Array
	DocumentationUrl	string
	ServicesPlanUrl		string
}


var _ = Describe("Application", func() {
	Before(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", serviceBrokerPath)).To(Say("App started"))
	})

	After(func() {
		Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
	})

	Describe("adding the broker", func() {
		It("adds the service broker to CF", func() {
			var appUri = "http://" + AppName + ".dijon.cf-apps.com"
			Expect(Cf("create-service-broker", AppName, "username", "password", appUri)).To(Say(
			 "Creating service broker test-broker as services...\nOK"))
		})
	})

	Describe("checking the catalog", func() {
		It("validates the catalog with CF", func() {
				var curlUri = "/v2/services"
				var output = Cf("curl", curlUri)

//				Expect(Cf("create-service-broker", AppName, "username", "password", appUri)).To(Say("OK"))
			})
	})
})


//we need a nyet that:
//registers a broker with a catalog
//validates that catalog is in ccdb but not public
//makes plans public
//validates that catalog is visible to end users
//modifies the catalog
//updates the broker
//validates changes are visible to end users
//deletes the broker
//validates that catalog is no longer in ccdb and is not visible to end users
