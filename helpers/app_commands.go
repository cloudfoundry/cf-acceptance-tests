package helpers

import (
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
)

func PushApp(appName string, appPath string) {
	push := Cf("push", appName, "-p", appPath)
	Expect(push).To(Say("App started"))
}

func DeleteApp(appName string) {
	delete := Cf("delete", appName, "-f")
	Expect(delete).To(Say("OK"))
}
