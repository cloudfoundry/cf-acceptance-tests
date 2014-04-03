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
The recommended way to do this is to run `go get github.com/cloudfoundry/cf-acceptance-tests`

All `go` dependencies required by CATs are vendored in `cf-acceptance-tests/Godeps`. The script `bin/test`
[ensures that](https://github.com/cloudfoundry/cf-acceptance-tests/blob/master/bin/test#L10-L15)
the vendored dependencies are available when executing the tests.

### Test Setup and Execution

To run the CF Acceptance tests, you will need a running CF instance; the examples below show the configuration
which would be used to run against a [bosh-lite](https://github.com/cloudfoundry/bosh-lite) installation.

The tests expect a set of credentials for an Admin and a regular user, an org, a temporary space,
and a persistent space within that org.

You must also provide a `$CONFIG` variable which points to a `.json` file that contains the application domain.

```bash
export ADMIN_USER=admin
export ADMIN_PASSWORD=admin
export CF_USER=cats-user
export CF_USER_PASSWORD=cats-user-pass
export CF_ORG=cats-org
export CF_SPACE=cats-space
export API_ENDPOINT=api.10.244.0.34.xip.io

gcf api $API_ENDPOINT
gcf auth $ADMIN_USER $ADMIN_PASSWORD
gcf create-user $CF_USER $CF_USER_PASSWORD
gcf create-org $CF_ORG
gcf create-space -o $CF_ORG $CF_SPACE
gcf target -o $CF_ORG -s $CF_SPACE
gcf set-space-role $CF_USER $CF_ORG $CF_SPACE SpaceManager
gcf set-space-role $CF_USER $CF_ORG $CF_SPACE SpaceDeveloper
gcf set-space-role $CF_USER $CF_ORG $CF_SPACE SpaceAuditor

gcf create-space persistent-space -o $CF_ORG
gcf set-space-role $CF_USER $CF_ORG persistent-space SpaceManager
gcf set-space-role $CF_USER $CF_ORG persistent-space SpaceDeveloper
gcf set-space-role $CF_USER $CF_ORG persistent-space SpaceAuditor

cat > integration_config.json <<EOF
{
  "apps_domain": "10.244.0.34.xip.io",
  "persistent_app_host": "persistent-app-6"
}
EOF
export CONFIG=$PWD/integration_config.json
```

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
