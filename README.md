# CF Acceptance Tests (CATs)

This test suite exercises a full [Cloud Foundry](https://github.com/cloudfoundry/cf-release) deployment using the golang `cf` CLI and `curl`. It is restricted to testing user-facing
features as a user interacting with the system via the CLI.

For example, one test pushes an app with `cf push`, hits an endpoint on the app with `curl` that causes it to crash, and asserts that we eventually see a crash event registered in `cf events`.

Tests that will NOT be introduced here are ones which could be tested at the component level,
such as basic CRUD of an object in the Cloud Controller. These tests belong with that component.

NOTE: Because we want to parallelize execution, tests should be written in such a way as to be runnable individually. This means that tests should not depend on state in other tests,
and should not modify the CF state in such a way as to impact other tests.

1. [Test Setup](#test-setup)  
    1. [Install Required Dependencies](#install-required-dependencies)  
    1. [Test Configuration](#test-configuration)
1. [Test Execution](#test-execution)
1. [Explanation of Test Suites](#explanation-of-test-suites)
1. [Contributing](#contributing)

## Test Setup

### Install Required Dependencies

Set up your golang development environment, per [golang.org](http://golang.org/doc/install).

You will probably also need the following SCM programs in order to `go get` source code:
* [git](http://git-scm.com/)
* [mercurial](http://mercurial.selenic.com/)
* [bazaar](http://bazaar.canonical.com/)

See [cf CLI](https://github.com/cloudfoundry/cli) for instructions on installing the go version of `cf`.

If you plan on running the tests in the `routing` package (not run by default), install the [Routing API CLI](https://github.com/cloudfoundry-incubator/routing-api-cli) v2.0.

Make sure that [curl](http://curl.haxx.se/) is installed on your system.

Make sure that the go version of `cf` is accessible in your `$PATH`.

Check out a copy of `cf-acceptance-tests` and make sure that it is added to your `$GOPATH`.
The recommended way to do this is to run `go get -d github.com/cloudfoundry/cf-acceptance-tests`. You will receive a warning "no buildable Go source files"; this can be ignored as there is no compilable go source code in the package, only test code.

All `go` dependencies required by CATs are vendored in `cf-acceptance-tests/Godeps`. The test script itself, [bin/test](https://github.com/cloudfoundry/cf-acceptance-tests/blob/master/bin/test), ensures that the vendored dependencies are available when executing the tests by prepending this directory to `$GOPATH`.

You will also of course need a running Cloud Foundry deployment to run these acceptance tess against.

### Test Configuration

You must set an environment variable `$CONFIG` which points to a JSON file that contains several pieces of data that will be used to configure the acceptance tests, e.g. telling the tests how to target your running Cloud Foundry deployment.

The following script will setup a sufficient `$CONFIG` to run the core test suites against a [BOSH-Lite](https://github.com/cloudfoundry/bosh-lite) deployment of CF.

```bash
#! /bin/bash

cat > integration_config.json <<EOF
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true
}
EOF
export CONFIG=$PWD/integration_config.json
```

The full set of config parameters is explained below:

* `api` (required): Cloud Controller API endpoint.
* `admin_user` (required): Name of a user in your CF instance with admin credentials.  This admin user must have the `doppler.firehose` scope if running the `logging` firehose tests.
* `admin_password` (required): Password of the admin user above.
* `apps_domain` (required): A shared domain that tests can use to create subdomains that will route to applications also craeted in the tests.
* `skip_ssl_validation`: Set to true if using an invalid (e.g. self-signed) cert for traffic routed to your CF instance; this is generally always true for BOSH-Lite deployments of CF.
* `system_domain` (only required for `routing` suite): Used to construct addresses for internal CF components, namely UAA and the Routing API, which are expected to live at `uaa.SYSTEM_DOMAIN` and `routing-api.SYSTEM_DOMAIN`.
* `client_secret` (only required for `routing` suite): Password used by gorouter to access the Routing API routes.
* `use_existing_user` (optional): The admin user configured above will normally be used to create a temporary user (with lesser permissions) to perform actions (such as push applications) during tests, and then delete said user after the tests have run; set this to `true` if you want to use an existing user, configured via the following properties.
* `keep_user_at_suite_end` (optional): If using an existing user (see above), set this to `true` unless you are okay having your existing user being deleted at the end. You can also set this to `true` when not using an existing user if you want to leave the temporary user around for debugging purposes after the test teardown.
* `existing_user` (optional): Name of the existing user to use.
* `existing_user_password` (optional): Password for the existing user to use.
* `persistent_app_host` (optional): [See below](#persistent-app-test-setup).
* `persistent_app_space` (optional): [See below](#persistent-app-test-setup).
* `persistent_app_org` (optional): [See below](#persistent-app-test-setup).
* `persistent_app_quota_name` (optional): [See below](#persistent-app-test-setup).
* `use_diego` (only required for `services` suite and CF deployments with Diego): Set to `true` if running `services` suite against a CF deployment with Diego.
* `artifacts_directory` (optional): If set, `cf` CLI trace output from test runs will be captured in files and placed in this directory. [See below](#capturing-test-output) for more.
* `default_timeout` (optional): Default time (in seconds) to wait for polling assertions that wait for asynchronous results.
* `cf_push_timeout` (optional): Default time (in seconds) to wait for `cf push` commands to succeed.
* `long_curl_timeout` (optional): Default time (in seconds) to wait for assertions that `curl` slow endpoints of test applications.
* `broker_start_timeout` (optional, only relevant for `services` suite): Time (in seconds) to wait for service broker test app to start.
* `test_password` (optional): Used to set the password for the test user. This may be needed if your CF installation has password policies.
* `timeout_scale` (optional): Used primarily to scale default timeouts for test setup and teardown actions (e.g. creating an org) as opposed to main test actions (e.g. pushing an app).
* `syslog_ip_address` (only required for `logging` suite): This must be a publically accessible IP address of your local machine, accessible by applications within your CF deployment.
* `syslog_drain_port` (only required for `logging` suite): This must be an available port on your local machine.
* `use_http` (optional): Set to true if you would like CF Acceptance Tests to use HTTP when making api and application requests. (defualt is HTTPS)

#### Persistent App Test Setup
The tests in `one_push_many_restarts_test.go` operate on an app that is supposed to persist between runs of the CF Acceptance tests. If these tests are run, they will create an org, space, and quota and push the app to this space. The test config will provide default names for these entities, but to configure them, set values for `persistent_app_host`, `persistent_app_space`, `persistent_app_org`, and `persistent_app_quota_name`.

#### Routing Test Suite Setup

The `routing` suite pushes applications which must be able to reach the load balancer of your Cloud Foundry deployment. This requires configuring application security groups to support this. Your deployment manifest should include the following data if you are running the `routing` suite:

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

#### Capturing Test Output
If you set a value for `artifacts_directory` in your `$CONFIG` file, then you will be able to capture `cf` trace output from failed test runs.  When a test fails, look for the node id and suite name ("*Applications*" and "*2*" in the example below) in the test output:

```bash
=== RUN TestLifecycle

Running Suite: Applications
====================================
Random Seed: 1389376383
Parallel test node 2/10. Assigned 14 of 137 specs.
```

The `cf` trace output for the tests in these specs will be found in `CF-TRACE-Applications-2.txt` in the `artifacts_directory`. 

### Test Execution

There are several different test suites, and you may not wish to run all the tests in all contexts, and sometimes you may want to focus individual test suites to pinpoint a failure.  The default set of tests can be run via:

```bash
./bin/test_default
```

This will run the `apps`, `internet_dependent`, and `security_groups` test suites, as well as the top level test suite that simply asserts that the installed `cf` CLI version is high enough to be compatible with the test suite.

For more flexibility you can run `./bin/test` and specify many more options, e.g. which suites to run, which suites to exclude (e.g. if you want to run all but one suite), whether or not to run the tests in parallel, the number of parallel nodes to use, etc.  Refer to [ginkgo documentation](http://onsi.github.io/ginkgo/) for full details.  

For example, to execute all test suites, and have tests run in parallel across four processes one would run:

```bash
./bin/test -r -nodes=4
```

Be careful with this number, as it's effectively "how many apps to push at once", as nearly every example pushes an app.

To execute the acceptance tests for a specific suite, e.g. `routing`, run the following:

```bash
bin/test routing
```

The suite names correspond to directory names.

To see verbose output from `ginkgo`, use the `-v` flag.

```bash
./bin/test routing -v
```

Most of these flags and options can also be passed to the `bin/test_default` script as well.

## Explanation of Test Suites

* The test suite in the top level directory of this repository simply asserts the the installed version of the `cf` CLI is compatible with the rest of the test suites.

* `apps`: Tests the core functionalities of Cloud Foundry: staging, running, logging, routing, buildpacks, etc.  This suite should always pass against a sound Cloud Foundry deployment.

* `internet_dependent`: This suite tests the feature of being able to specify a buildpack via a Github URL.  As such, this depends on your Cloud Foundry application containers having access to the Internet.  You should take into account the configuration of the network into which you've deployed your Cloud Foundry, as well as any security group settings applied to application containers.

* `logging`: This test exercises the syslog drain forwarding functionality. A TCP listener is
spun up on the running machine, an app is deployed to the target Cloud Foundry
and bound to that listener (as a syslog drain) and the drain is checked for log
messages.  Tests in this package are only intended to be run on machines that are accessible by your deployment.

* `operator`: Tests in this package are only intended to be run in non-production environments.  They may not clean up after themselves and may affect global CF state.  They test some miscellaneous features; read the tests for more details.

* `routing`: This package contains routing specific acceptance tests, for example: Routing API (register, unregister, list, server-sent events), Route Services, and GoRouter (Context path, wildcard, SSL termination, sticky sessions).  At the time of this writing, many of the routing features are works in progress.

* `security_groups`: This suite tests the security groups feature of Cloud Foundry that lets you apply rules-based controls to network traffic in and out of your containers.  These should pass for most recent Cloud Foundry installations.  `cf-release` versions `v200` and up should have support for most security group specs to pass.

* `services`: This suite tests various features related to services, e.g. registering a service broker via the service broker API.  Some of these tests exercise special integrations, such as Single Sign-On authentication; you may wish to run some tests in this package but selectively skip others if you haven't configured the required integrations.  Consult the [ginkgo spec runner](http://onsi.github.io/ginkgo/#the-spec-runner) documention to see how to use the `--skip` and `--focus` flags.

* `v3`: This suite contains tests for the next-generation v3 Cloud Controller API.  As of this writing, the v3 API is not officially supported.


## Contributing

This repository uses [godep](https://github.com/tools/godep) to manage `go` dependencies.

All `go` packages required to run these tests are vendored into the `cf-acceptance-tests/Godeps` directory.

When making changes to the test suite that bring in additional `go` packages, you should use the workflow described in the [Add or Update a Dependency](https://github.com/tools/godep#add-a-dependency) section of the godep documentation.
