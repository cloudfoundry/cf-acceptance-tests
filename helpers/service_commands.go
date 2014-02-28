package helpers

import (
	"time"
	"encoding/json"
	"os"

	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func MakePlanPublic(url string) {
	jsonMap := make(map[string]bool)
	jsonMap["public"] = true
	planJson, _ := json.Marshal(jsonMap)
	Expect(Cf("curl", url, "-X", "PUT", "-d", string(planJson))).To(ExitWithTimeout(0, 5*time.Second))
}

type RequireExpectation struct {
	actual interface{}
}

func Require(actual interface{}) RequireExpectation {
	return RequireExpectation {
		actual: actual,
	}
}

func (e RequireExpectation) To(matcher OmegaMatcher, optionalDescription ...interface{}) {
	result := Expect(e.actual).To(matcher, optionalDescription...)
	if !result {
		panic("Failed Required Expectation")
	}
}

func Recover() {
	if r := recover(); r != nil {
		if r != "Failed Required Expectation" {
			panic(r)
		}
	}
}

func LoginAsAdmin() {
	Expect(Cf("login", "-u", os.Getenv("ADMIN_USER"), "-p", os.Getenv("ADMIN_PASSWORD"), "-o", os.Getenv("CF_ORG"), "-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}

func LoginAsUser() {
	Expect(Cf("login", "-u", os.Getenv("CF_USER"), "-p", os.Getenv("CF_USER_PASSWORD"), "-o", os.Getenv("CF_ORG"), "-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}
