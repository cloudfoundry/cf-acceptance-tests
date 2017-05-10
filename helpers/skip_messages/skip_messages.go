package skip_messages

const SkipAppsMessage string = `Skipping this test because config.IncludeApps is set to 'false'.`
const SkipBackendCompatibilityMessage string = `Skipping this test because config.IncludeBackendCompatibility is set to 'false'.
NOTE: Ensure that your platform is running both DEA and Diego before running this test.`
const SkipContainerNetworkingMessage string = `Skipping this test because Config.IncludeContainerNetworking is set to 'false'.`
const SkipDeaMessage string = `Skipping this test because Config.Backend is not set to 'dea'.
NOTE: Ensure that your platform is running DEAs before enabling this test.`
const SkipDetectMessage string = `Skipping this test because config.IncludeDetect is set to 'false'.`
const SkipDiegoMessage string = `Skipping this test because Config.Backend is not set to 'diego'.
NOTE: Ensure that your platform is running Diego before enabling this test.`
const SkipDockerMessage string = `Skipping this test because config.IncludeDocker is set to 'false'.
NOTE: Ensure Docker containers are enabled on your platform before enabling this test.`
const SkipInternetDependentMessage string = `Skipping this test because config.IncludeInternetDependent is set to 'false'.
NOTE: Ensure that your platform has access to the internet before running this test.`
const SkipPrivateDockerRegistryMessage string = `Skipping this test because config.IncludePrivateDockerRegistry is set to 'false'.
NOTE: Ensure that you've provided values for config.PrivateDockerRegistryImage, config.PrivateDockerRegistryUsername,
and config.PrivateDockerRegistryPassword before running this test.`
const SkipPersistentAppMessage string = `Skipping this test because config.IncludePersistentApp is set to 'false'.`
const SkipPrivilegedContainerSupportMessage string = `Skipping this test because Config.IncludePrivilegedContainerSupport is set to 'false'.
NOTE: Ensure privileged containers are allowed on your platform before enabling this test.`
const SkipRouteServicesMessage string = `Skipping this test because config.IncludeRouteServices is set to 'false'.
NOTE: Ensure that route services are enabled on your platform before running this test.`
const SkipRoutingMessage string = `Skipping this test because config.IncludeRouting is set to 'false'.`
const SkipSecurityGroupsMessage string = `Skipping this test because config.IncludeSecurityGroups is set to 'false'.
NOTE: Ensure that your platform restricts internal network traffic by default in order to run this test.`
const SkipServicesMessage string = `Skipping this test because config.IncludeServices is set to 'false'.`
const SkipSSHMessage string = `Skipping this test because config.IncludeSsh is set to 'false'.
NOTE: Ensure that your platform is deployed with a Diego SSH proxy in order to run this test.`
const SkipSSOMessage string = `Skipping this test because config.IncludeSSO is not set to 'true'.
NOTE: Ensure that your platform is running UAA with SSO enabled before enabling this test.`
const SkipTasksMessage string = `Skipping this test because config.IncludeTasks is set to 'false'.
NOTE: Ensure tasks are enabled on your platform before enabling this test.`
const SkipV3Message string = `Skipping this test because config.IncludeV3 is set to 'false'.
NOTE: Ensure that the v3 api features are enabled on your platform before running this test.`
