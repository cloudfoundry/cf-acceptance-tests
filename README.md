# CF Acceptance Tests (CATs)

This test suite exercises a full [Cloud Foundry](https://github.com/cloudfoundry/cf-release) deployment using
the golang `cf` CLI and `curl`. It is restricted to testing user-facing
features as a user interacting with the system via the CLI.

For example, one test pushes an app with `cf push`, hits an endpoint on the
app with `curl` that causes it to crash, and asserts that we eventually see a
crash event registered in `gcf events`.

Tests that will NOT be introduced here are ones which could be tested at the component level,
such as basic CRUD of an object in the Cloud Controller. These tests belong with that component.

NOTE: Because we want to parallize execution, tests should be written in such a way as to be runable individually.
This means that tests should not depend on state in other tests,
and should not modify the CF state in such a way as to impact other tests.

## Running the tests

### Set up your `go` environment

Set up your golang development environment, [per golang.org](http://golang.org/doc/install).

You will probably also need the following SCM programs in order to `go get` source code:
* [git](http://git-scm.com/)
* [mercurial](http://mercurial.selenic.com/)
* [bazaar](http://bazaar.canonical.com/)

See [Go CLI](https://github.com/cloudfoundry/cli) for instructions on installing the go version of `cf`.

Make sure that [curl](http://curl.haxx.se/) is installed on your system.

Make sure that the go version of `cf` is accessible in your `$PATH`, and that it is either
renamed to `gcf`, or that there is a symlink from `gcf` to the location of `cf`.

Check out a copy of `cf-acceptance-tests` and make sure that it is added to your `$GOPATH`.
The recommended way to do this is to run `go get github.com/cloudfoundry/cf-acceptance-tests`. You will receive a
warning "no buildable Go source files"; this can be ignored as there is no compilable go code in the package.

All `go` dependencies required by CATs are vendored in `cf-acceptance-tests/Godeps`. The test script itself, `bin/test`,
[ensures that](https://github.com/cloudfoundry/cf-acceptance-tests/blob/master/bin/test#L10-L15)
the vendored dependencies are available when executing the tests by prepending this directory to `$GOPATH`.

### Test Setup

To run the CF Acceptance tests, you will need:
- a running CF instance
- credentials for an Admin user
- an environment variable `$CONFIG` which points to a `.json` file that contains the application domain

The following script will configure these prerequisites for a [bosh-lite](https://github.com/cloudfoundry/bosh-lite)
installation. Replace credentials and URLs as appropriate for your environment.

```bash
#! /bin/bash

cat > integration_config.json <<EOF
{
  "api": "api.10.244.0.34.xip.io",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "10.244.0.34.xip.io"
}
EOF
export CONFIG=$PWD/integration_config.json
```

If you are running the tests with version 6.0.2 or later of the Go CLI against bosh-lite or any other environment
using self-signed certificates, add

```
  "skip_ssl_validation": true
```

to your integration_config.json as well.


### Persistent App Test Setup

The tests in `one_push_many_restarts_test.go` operate on an app that is supposed to persist between runs of the CF
Acceptance tests. If these tests are run, they will create an org, space, and quota and push the app to this space.
The test config will provide default names for these entities, but to configure them, add the following key-value
pairs to integration_config.json:

```
  "persistent_app_host": "myapp",
  "persistent_app_space": "myspace",
  "persistent_app_org": "myorg",
  "persistent_app_quota_name": "myquota",
```

### Test Execution

To execute the tests, run:

```bash
./bin/test
```

Internally the `bin/test` script runs tests using [ginkgo](https://github.com/onsi/ginkgo).

Arguments, such as `-focus=`, `-nodes=`, etc., that are passed to the script are sent to `ginkgo`

For example, to execute tests in parallel across four processes one would run:

```bash
./bin/test -nodes=4
```

Be careful with this number, as it's effectively "how many apps to push at once", as nearly every example pushes an app.

#### Seeing command-line output

To see verbose output from `cf`, set `CF_VERBOSE_OUTPUT` to `true` before running the tests.

```bash
export CF_VERBOSE_OUTPUT=true
```

#### Capturing CF CLI output

Set `CF_TRACE_BASENAME` and the trace output from `cf` will be captured in files named
`${CF_TRACE_BASENAME}${Ginkgo Node Id}.txt`.

```bash
export CF_TRACE_BASENAME=cf_trace_
```

The following files may be created:

```bash
cf_trace_1.txt
cf_trace_2.txt
...
```

If a test fails, look for the node id is the test output:

```bash
=== RUN TestLifecycle

Running Suite: Application Lifecycle
====================================
Random Seed: 1389376383
Parallel test node 2/10. Assigned 14 of 137 specs.
```

The `cf` trace output for the tests in these specs will be found in `cf_trace_2.txt`


## Changing CATs

### Dependency Management

CATs use [godep](https://github.com/tools/godep) to manage `go` dependencies.

All `go` packages required to run CATs are vendored into the `cf-acceptance-tests/Godeps` directory.

When making changes to the test suite that bring in additional `go` packages, you should use the workflow described in the
[Add or Update a Dependency](https://github.com/tools/godep#add-or-update-a-dependency) section of the godep documentation.
