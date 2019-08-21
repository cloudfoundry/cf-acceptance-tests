# CF Acceptance Tests (CATs)
This suite exercises a [Cloud Foundry](https://github.com/cloudfoundry/cf-deployment)
deployment using the `cf` CLI and `curl`.
It is scoped to testing user-facing,
end-to-end features.

For example,
one test pushes an app with `cf push`,
hits an endpoint on the app with `curl`
that causes it to crash,
and asserts that we see
a crash event in `cf events`.

Tests that _won't_ be introduced here
include things like basic CRUD of an object in the Cloud Controller.
Such tests belong with the component they relate to.

These tests are not intended for use against production systems.
They're meant for acceptance environments
used by people developing Cloud Foundry's releases.
While these tests attempt to clean up after themselves,
there's no guarantee that they won't
change the state of your system in an undesirable way.
For lightweight system tests that are safe to run against a production environment,
please use the [CF Smoke Tests](https://github.com/cloudfoundry/cf-smoke-tests).

**NOTE:** Because we want to parallelize execution,
tests should be written to be executable independently.
Tests should not depend on state in other tests,
and should not modify the CF state
in such a way as to impact other tests.

1. [Test Setup](#test-setup)
    1. [Install Required Dependencies](#install-required-dependencies)
    1. [Test Configuration](#test-configuration)
1. [Test Execution](#test-execution)
1. [Explanation of Test Groups](#explanation-of-test-groups)
1. [Contributing](#contributing)

## Test Setup
### Prerequisites for running CATS
- Install golang >= `1.11`. Set up your golang development environment, per
  [golang.org](http://golang.org/doc/install).
- Install the [`cf CLI`](https://github.com/cloudfoundry/cli).
  Make sure that it is accessible in your `$PATH`.
- Install [curl](http://curl.haxx.se/)
- Check out a copy of `cf-acceptance-tests`. It uses Go modules, so there is no need to put it in `$GOPATH`.
- Ensure all submodules are checked out to the correct SHA.
  The easiest way to do this is by running:

  ```bash
  ./bin/update_submodules
  ```
- Install a running Cloud Foundry deployment
  to run these acceptance tests against.
  For example, bosh-lite.

### Updating `go` dependencies
All `go` dependencies required by CATs
are vendored in the `vendor` directory.

Make sure to have Golang 1.11+

In order to update a current dependency to a specific version,
do the following:

```bash
cd cf-acceptance-tests
source .envrc
go get <import_path>@<revision_number>
go mod vendor
```

If you'd like to add a new dependency just run:

```bash
go mod tidy
go mod vendor
```

## Test Configuration
You must set the environment variable `$CONFIG`
which points to a JSON file
that contains several pieces of data
that will be used to configure the acceptance tests,
e.g. telling the tests how to target
your running Cloud Foundry deployment
and what tests to run.

The following can be pasted into a terminal
and will set up a sufficient `$CONFIG`
to run the core test suites
against a [BOSH-Lite](https://github.com/cloudfoundry/bosh-lite)
deployment of CF.

See [`example-cats-config.json`](example-cats-config.json) for reference.

Only the following test groups are run by default:
```
include_apps
include_detect
include_routing
include_v3
include_capi_no_bridge
```

#### The full set of config parameters is explained below:
##### Required parameters:
* `api`: Cloud Controller API endpoint.
* `admin_user`: Name of a user in your CF instance with admin credentials.  This admin user must have the `doppler.firehose` scope.
* `admin_password`: Password of the admin user above.
* `apps_domain`: A shared domain that tests can use to create subdomains that will route to applications also created in the tests.
* `skip_ssl_validation`: Set to true if using an invalid (e.g. self-signed) cert for traffic routed to your CF instance; this is generally always true for BOSH-Lite deployments of CF.

##### Optional parameters:
`include_*` parameters are used to specify whether to skip tests based on how a deployment is configured.
* `include_apps`: Flag to include the apps test group.
* `include_backend_compatibility`: Flag to include whether we check DEA/Diego interoperability.
* `include_container_networking`: Flag to include tests related to container networking.
  `include_security_groups` must also be set for tests to run. [See below](#container-networking-and-application-security-groups)
* `credhub_mode`: Valid values are `assisted` or `non-assisted`. [See below](#credhub-modes).
* `credhub_location`: Location of CredHub instance; default is `https://credhub.service.cf.internal:8844`
* `credhub_client`: UAA client credential for Service Broker write access to CredHub (required for CredHub tests); default is `credhub_admin_client`.
* `credhub_secret`: UAA client secret for Service Broker write access to CredHub (required for CredHub tests).
* `include_capi_no_bridge`: Flag to run tests that require CAPI's (currently optional) bridge consumption features.
* `include_deployments`: Flag to include tests for the cloud controller rolling deployments. V3 must also be enabled.
* `include_detect`: Flag to include tests in the detect group.
* `include_docker`: Flag to include tests related to running Docker apps on Diego. Diego must be deployed and the CC API docker_diego feature flag must be enabled for these tests to pass.
* `include_internet_dependent`: Flag to include tests that require the deployment to have internet access.
* `include_internetless`: Flag to include tests that require the deployment to not have internet access.
* `include_isolation_segments`: Flag to include isolation segment tests.
* `include_private_docker_registry`: Flag to run tests that rely on a private docker image. [See below](#private-docker).
* `include_route_services`: Flag to include the route services tests. Diego must be deployed for these tests to pass.
* `include_routing`: Flag to include the routing tests.
* `include_routing_isolation_segments`: Flag to include routing isolation segments tests. [See below](#routing-isolation-segments). Cannot be run together with logging isolation segments tests.
* `include_security_groups`: Flag to include tests for security groups. [See below](#container-networking-and-application-security-groups)
* `include_services`: Flag to include test for the services API.
* `include_service_instance_sharing`: Flag to include tests for service instance sharing between spaces. `include_services` must be set for these tests to run. The `service_instance_sharing` feature flag must also be enabled for these tests to pass.
* `include_ssh`: Flag to include tests for Diego container ssh feature.
* `include_sso`: Flag to include the services tests that integrate with Single Sign On. `include_services` must also be set for tests to run.
* `include_tasks`: Flag to include the v3 task tests. `include_v3` must also be set for tests to run. The CC API task_creation feature flag must be enabled for these tests to pass.
* `include_tcp_routing`: Flag to include the TCP Routing tests. These tests are equivalent to the [TCP Routing tests](https://github.com/cloudfoundry/routing-acceptance-tests/blob/master/tcp_routing/tcp_routing_test.go) from the Routing Acceptance Tests.
* `include_v3`: Flag to include tests for the v3 API.
* `include_zipkin`: Flag to include tests for Zipkin tracing. `include_routing` must also be set for tests to run. CF must be deployed with `router.tracing.enable_zipkin` set for tests to pass.
* `use_http`: Set to true if you would like CF Acceptance Tests to use HTTP when making api and application requests. (default is HTTPS)
* `use_log_cache`: Set to false if you don't want CF Acceptance Tests to use Log Cache for reading application logs. (default is true)
* `use_existing_organization`: Set to true when you need to specify an existing organization to use rather than creating a new organization.
* `existing_organization`: Name of the existing organization to use.
* `use_existing_user`: The admin user configured above will normally be used to create a temporary user (with lesser permissions) to perform actions (such as push applications) during tests, and then delete said user after the tests have run; set this to `true` if you want to use an existing user, configured via the following properties.
* `keep_user_at_suite_end`: If using an existing user (see above), set this to `true` unless you are okay having your existing user being deleted at the end. You can also set this to `true` when not using an existing user if you want to leave the temporary user around for debugging purposes after the test teardown.
* `existing_user`: Name of the existing user to use.
* `existing_user_password`: Password for the existing user to use.
* `artifacts_directory`: If set, `cf` CLI trace output from test runs will be captured in files and placed in this directory. [See below](#capturing-test-output) for more.
* `default_timeout`: Default time (in seconds) to wait for polling assertions that wait for asynchronous results.
* `cf_push_timeout`: Default time (in seconds) to wait for `cf push` commands to succeed.
* `long_curl_timeout`: Default time (in seconds) to wait for assertions that `curl` slow endpoints of test applications.
* `broker_start_timeout` (only relevant for `services` test group): Time (in seconds) to wait for service broker test app to start.
* `async_service_operation_timeout` (only relevant for the `services` test group): Time (in seconds) to wait for an asynchronous service operation to complete.
* `test_password`: Used to set the password for the test user. This may be needed if your CF installation has password policies.
* `timeout_scale`: Used primarily to scale default timeouts for test setup and teardown actions (e.g. creating an org) as opposed to main test actions (e.g. pushing an app).
* `isolation_segment_name`: Name of the isolation segment to use for the isolation segments test.
* `isolation_segment_domain`: Domain that will route to the isolated router in the isolation segments and routing isolation segments tests. [See below](#routing-isolation-segments)
* `private_docker_registry_image`: Name of the private docker image to use when testing private docker registries. [See below](#private-docker)
* `private_docker_registry_username`: Username to access the private docker repository. [See below](#private-docker)
* `private_docker_registry_password`: Password to access the private docker repository. [See below](#private-docker)
* `unallocated_ip_for_security_group`: An unused IP address in the private network used by CF. Defaults to 10.0.244.255. [See below](#container-networking-and-application-security-groups)

* `require_proxied_app_traffic`: Set this to `true` if Diego was configured to require proxied port mappings, i.e. if `containers.proxy.enable_unproxied_port_mappings` is set to `false`.  Note that this also requires using the [cf-syslog-skip-cert-verify](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/cf-syslog-skip-cert-verify.yml).

* `staticfile_buildpack_name` [See below](#buildpack-names)
* `java_buildpack_name` [See below](#buildpack-names)
* `ruby_buildpack_name` [See below](#buildpack-names)
* `nodejs_buildpack_name` [See below](#buildpack-names)
* `go_buildpack_name` [See below](#buildpack-names)
* `python_buildpack_name` [See below](#buildpack-names)
* `php_buildpack_name` [See below](#buildpack-names)
* `binary_buildpack_name` [See below](#buildpack-names)

* `include_windows`: Flag to include the tests that run against Windows cells.
* `use_windows_test_task`: Flag to include the tasks tests on Windows cells. Default is `false`.
* `use_windows_context_path`: Flag to include the Windows context path routing tests. Default is `false`.
* `windows_stack`: Windows stack to run tests against. Must be either `windows2012R2`, or `windows`. Defaults to `windows2012R2`.

* `include_service_discovery`: Flag to include test for the service discovery. These tests use `apps.internal` domain, which is the default in `cf-networking-release`. The internal domain is currently not configurable.
* `stacks`: An array of stacks to test against. Currently only `cflinuxfs3` stack is supported. Default is `[cflinuxfs3]`.

* `include_volume_services`: Flag to include the tests for volume services. The following requirements must be met to run this suite: Diego must be deployed. Docker support must be enabled. tcp-routing must be deployed. 
* `volume_service_name`: The name of the volume service provided by the volume service broker.
* `volume_service_plan_name`: The name of the plan of the service provided by the volume service broker.
* `volume_service_create_config`: The JSON configuration that is used when volume service is created.

#### Buildpack Names
Many tests specify a buildpack when pushing an app, so that on diego the app staging process completes in less time. The default names for the buildpacks are as follows; if you have buildpacks with different names, you can override them by setting different names:

* `staticfile_buildpack_name: staticfile_buildpack`
* `java_buildpack_name: java_buildpack`
* `ruby_buildpack_name: ruby_buildpack`
* `nodejs_buildpack_name: nodejs_buildpack`
* `go_buildpack_name: go_buildpack`
* `python_buildpack_name: python_buildpack`
* `php_buildpack_name: php_buildpack`
* `binary_buildpack_name: binary_buildpack`
* `hwc_buildpack_name: hwc_buildpack`

#### Route Services Test Group Setup
The `route_services` test group pushes applications which must be able to reach the load balancer of your Cloud Foundry deployment. This requires configuring application security groups to support this. Your deployment manifest should include the following data if you are running the `route_services` group:

```yaml
...
properties:
  ...
  cc:
    ...
    security_group_definitions:
      - name: load_balancer
        rules:
        - protocol: all
          destination: IP_OF_YOUR_LOAD_BALANCER # (e.g. 10.244.0.34 for a standard deployment of Cloud Foundry on BOSH-Lite)
    default_running_security_groups: ["load_balancer"]
```

#### Container Networking and Application Security Groups
To run tests that exercise container networking and running application security groups, the `include_security_groups` flags must be true.

The Windows ASG tests require an unallocated IP
on the private network used by the CF deployment,
set with the `unallocated_ip_for_security_group` config value.
Environments created by bbl on public clouds
can use the default value of 10.0.244.255.
vSphere and Openstack environments may require a custom IP.

#### Private Docker
To run tests that exercise the use of credentials to access a private docker registry, the `include_private_docker_registry` flag must be true, and the following config values must be provided:

* `private_docker_registry_image`
* `private_docker_registry_username`
* `private_docker_registry_password`

These tests assume that the specified private docker image is a private version of the cloudfoundry/diego-docker-app-custom:latest. To upload a private version to your DockerHub account, first create a private repository on DockerHub and log in to docker on the command line. Then run the following commands:

```bash
docker pull cloudfoundry/diego-docker-app-custom:latest
docker tag cloudfoundry/diego-docker-app-custom:latest <your-private-repo>:<some-tag>
docker push <your-private-repo>:<some-tag>
```

The value for the `private_docker_registry_image` config value in this case would be "<your-private-repo>:<some-tag>".

#### Routing Isolation Segments
To run tests that involve routing isolation segments, the following config values must be provided:
* `isolation_segment_name`
* `isolation_segment_domain`

Read the documentation [here](http://docs.cloudfoundry.org/adminguide/routing-is.html) for further setup details.

#### Credhub Modes
- `non-assisted` mode means that apps are responsible for resolving Credhub refs for credentials.
  When the user binds a service to an app, the service broker will store a credential in Credhub and pass the ref back to the Cloud Controller.
  When the user restages the app, the Cloud Controller will pass the Credhub ref to the app in the `VCAP_SERVICES` environment variable,
  at which point the app can make a request directly to Credhub to resolve the ref and obtain the credential.
  This mode is enabled when `cc.credential_references.interpolate_service_bindings` is false -- which is the non-default configuration.
- `assisted` mode means that the Credhub ref will be resolved before the app starts running.
  As before, when the user binds a service to an app, the service broker will store a credential in Credhub and pass the ref back to the Cloud Controller.
  This time, when the user restages the app, the Cloud Controller will pass the Credhub ref to the Diego runtime,
  at which point the launcher (from the buildpackapplifecycle or the dockerapplifecycle components) will resolve the Credhub ref,
  and store the credential in `VCAP_SERVICES` for the app to consume.
  This mode is enabled when `cc.credential_references.interpolate_service_bindings` is true -- which is the default configuration.

#### Capturing Test Output
When a test fails, look for the test group name (`[services]` in the example below) in the test output:

```bash
• Failure in Spec Setup (BeforeEach) [34.662 seconds]
[services] Service Instance Lifecycle
```

If you set a value for `artifacts_directory` in your `$CONFIG` file, then you will be able to capture `cf` trace output from failed test runs, this output may be useful in cases where the normal test output is not enough to debug an issue.  The `cf` trace output for the tests in these specs will be found in `CF-TRACE-Applications-*.txt` in the `artifacts_directory`.

## Test Execution
To execute all test groups, run the following from the root directory of cf-acceptance-tests:
```bash
./bin/test
```

##### Parallel execution
It's possible execute all test groups, and have tests run in parallel across multiple processes.
This parallelization can significantly decrease CATs runtime.

_However_, be careful with this number, as it's effectively "how many apps to push at once", as nearly every example pushes an app.

In order to run 12 processes in parallel run the following:
```bash
./bin/test -nodes=12
```

For reference here's how many parallel processes the Release Integration team runs with:

| Foundation Type | # of Parallel Processes |
| ----------- | ----------- |
| [Vanilla CF](https://github.com/cloudfoundry/cf-deployment/blob/master/cf-deployment.yml) | 12 |
| [BOSH Lite](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/bosh-lite.yml) | 4 |


##### Focusing Test Groups
If you are already familiar with CATs you probably know that there are many test groups. You may not wish to run all the tests in all contexts, and sometimes you may want to focus individual test groups to pinpoint a failure. To execute a specific group of acceptance tests, e.g. `routing/`, edit your [`integration_config.json`](#test-configuration) file and set all `include_*` values to `false` except for `include_routing` then run the following:

```bash
./bin/test
```

To execute tests in a single file use an `FDescribe` block around the tests in that file:
```go
var _ = BackendCompatibilityDescribe("Backend Compatibility", func() {
  FDescribe("Focused tests", func() { // Add this line here
  // ... rest of file
  }) // Close here
})

```

The test group names correspond to directory names.

##### Verbose Output
To see verbose output from `ginkgo`, use the `-v` flag.

```bash
./bin/test -v
```

You can of course combine the `-v` flag with the `-nodes=N` flag.

## Explanation of Test Groups

Test Group Name| Description
--- | ---
`apps`| Tests the core functionalities of Cloud Foundry: staging, running, logging, routing, buildpacks, etc.  This test group should always pass against a sound Cloud Foundry deployment.
`backend_compatibility` | Tests interoperability of droplets staged on the DEAs running on Diego
`credhub`| Tests CredHub-delivered Secure Service credentials in the service binding. [CredHub configuration][credhub-secure-service-credentials] is required to run these tests. In addition to selecting a `credhub_mode`, `credhub_client` and `credhub_secret` values are required for these tests.
`detect` | Tests the ability of the platform to detect the correct buildpack for compiling an application if no buildpack is explicitly specified.
`docker`| Tests our ability to run docker containers on Diego and that we handle docker metadata correctly.
`internet_dependent`| Tests the feature of being able to specify a buildpack via a Github URL.  As such, this depends on your Cloud Foundry application containers having access to the Internet.  You should take into account the configuration of the network into which you've deployed your Cloud Foundry, as well as any security group settings applied to application containers.
`internetless`| Tests that your Cloud Foundry application containers do not have access to the Internet.  You should take into account the configuration of the network into which you've deployed your Cloud Foundry, as well as any security group settings applied to application containers.
`isolation_segments` | This test group requires that Diego be deployed with a minimum of 2 cells. One of those cells must have been deployed with a `placement_tag`. If the deployment has been deployed with a routing isolation segment, `isolation_segment_domain` must also be set. For more information, please refer to the [Isolation Segments documentation](https://docs.cloudfoundry.org/adminguide/isolation-segments.html).
`route_services` | Tests the [Route Services](https://docs.cloudfoundry.org/services/route-services.html) feature of Cloud Foundry.
`routing`| This package contains routing specific acceptance tests (context paths, wildcards, SSL termination, sticky sessions, and zipkin tracing).
`routing_isolation_segments` | Tests that requests to isolated apps are only routed through isolated routers, and vice versa. It requires all of the setup for the isolation segments test suite. Additionally, a minimum of two Gorouter instances must be deployed. One instance must be configured with the property `routing_table_sharding_mode: shared-and-segments`. The other instance must have the properties `routing_table_sharding_mode: segments` and `isolation_segments: [YOUR_PLACEMENT_TAG_HERE]`. The `isolation_segment_name` in the CATs properties must match the `placement_tag` and `isolation_segment`.`isolation_segment_domain` must be set and traffic to that domain should go to the isolated router.
`security_groups`| Tests the [Security Groups](https://docs.cloudfoundry.org/concepts/asg.html) feature of Cloud Foundry.
`service_discovery`| Tests the [Service Discovery](https://docs.cloudfoundry.org/devguide/deploy-apps/cf-networking.html#discovery) feature for applications running on Cloud Foundry.
`services`| Tests various features related to services, e.g. registering a service broker via the service broker API.  Some of these tests exercise special integrations, such as Single Sign-On authentication; you may wish to run some tests in this package but selectively skip others if you haven't configured the required integrations (by setting `include_sso` parameter to `false`in your configuration).
`ssh`| Tests communication with Diego apps via ssh, scp, and sftp.
`tasks`| Tests Cloud Foundry's [Tasks](https://docs.cloudfoundry.org/devguide/using-tasks.html) feature.
`tcp_routing`| Tests TCP Routing Feature of Cloud Foundry. You need to make sure you've set up a TCP domain `tcp.<SYSTEM_DOMAIN>` as described [here](https://docs.cloudfoundry.org/adminguide/enabling-tcp-routing.html). If you are using `bbl` (BOSH Bootloader), TCP domain is set up for you automatically.
`v3`| This test group contains tests for the next-generation v3 Cloud Controller API.
`volume_services` | Tests the [Volume Services](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html) feature of Cloud Foundry.

## Contributing

This repository uses [go mod](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more)
to manage `go` dependencies.

All `go` dependencies required by CATs are vendored in the `vendor` directory.

When making changes to the test suite that bring in additional `go` packages,
you should use the following workflow:

If you can use the most recent version of a dependency use `go mod tidy`,
otherwise use `go get <dependency>@<version>`. Both of these require go modules
to be enabled via the [envrc](.envrc). Finally use `go mod vendor` to add the
dependencies to the `vendor` directory.

For tools and assets, please use the [helpers/assets/tools.go](helpers/assets/tools.go) via
the [go mod tool workflow](https://github.com/go-modules-by-example/index/tree/master/010_tools)

For additional information refer to the [official
wiki](https://github.com/golang/go/wiki/Modules) and the [official examples
repo](https://github.com/go-modules-by-example/index)

Although the default branch for this repository is `master`, we ask that all
pull requests be made against the `develop` branch. Please run the unit tests
and make sure they are passing before submitting. Use `./bin/run_units` to run
these unit tests.

**Note**: it is necessary to run the tests from the root of the repo.

### Code Conventions

There are a number of conventions we recommend developers of CF acceptance tests
adopt:

1. When pushing an app:
  * set the **memory** requirement, and use the suite's `DEFAULT_MEMORY_LIMIT` (`DEFAULT_WINDOWS_MEMORY_LIMIT` for tests in the `windows` directory) unless the test specifically needs to test a different value,
  * set the **buildpack** unless the test specifically needs to test the case where a buildpack is unspecified, and use one of `config.RubyBuildpack`, `config.JavaBuildpack`, etc.
unless the test specifically needs to use a buildpack name or URL specific to the test,
  * set the **domain**, and use the `Config.AppsDomain` unless the test specifically needs to test a different app domain.

  For example:

  ```go
  Expect(cf.Cf("push", appName,
      "-b", buildpackName,                  // specify buildpack
      "-m", DEFAULT_MEMORY_LIMIT,           // specify memory limit
      "-d", Config.AppsDomain,              // specify app domain
  ).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
  ```
1. Delete all resources that are created, e.g. apps, routes, quotas, etc.  This is in order to leave the system in the same state it was found in.  For example, to delete apps and their associated routes:
    ```
		Expect(cf.Cf("delete", myAppName, "-f", "-r").Wait()).To(Exit(0))
    ```
1. Specifically for apps, before tearing them down, print the app guid and recent application logs. There is a helper method `AppReport` provided in the `app_helpers` package for this purpose.

    ```go
    AfterEach(func() {
      app_helpers.AppReport(appName)
    })
    ```
1. Document the purpose of your test groups in this repo's README.md.  This is especially important when changing the explicit behavior of existing test groups or adding new test groups.
1. Document all changes to the config object in this repo's README.md.
1. If you add a test that requires a new minimum `cf` CLI version, update the `minCliVersion` in `cats_suite_test.go` .

[networking-releases]: https://github.com/cloudfoundry-incubator/cf-networking-release/releases
[credhub-secure-service-credentials]: https://github.com/pivotal-cf/credhub-release/blob/master/docs/secure-service-credentials.md
