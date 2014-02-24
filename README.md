# CF Acceptance Tests

This test suite exercises a full [Cloud Foundry][cf-release] deployment using
the `gcf` CLI and `curl`. It is restricted to testing high-level user-facing
features that touch more than one component in the deployment.

For example, one test pushes an app with `gcf push`, hits an endpoint on the
app with `curl` that causes it to crash, and asserts that we eventually see a
crash event registered in `gcf events`.

Tests that will NOT be introduced here are things that only really test one
single component, like basic CRUD of an object in the CC. These tests belong
further down.

When running the tests, they will use `gcf` in whatever state it's already
configured for. Setting up users and organizations and spaces is out of scope.
This simplifies the tests themselves and is likely more natural for someone
testing their own CF deployment, as they're probably kicking the tires on it
themselves.

## Running the tests

You will need a working Go environment with `$GOPATH` set, and you will need
`gcf` and `curl` in your `$PATH`.

See [Go CLI][cli] for instructions on installing `gcf`. See [Go][go] for
instructions on installing `go`.

### Configuration

Before running the tests, you must make sure you've logged in to your
runtime environment and targetted a space using 
```
  gcf target -o [your_org] -s [your_space]
```

You must also set `$CONFIG` to point to a `.json` file which contains the 
configuration for the tests.

There is not much to configure - for now you just need to set the domain that
the deployment is configured to use for the apps.

For example, to run these against a local BOSH lite deployment, you'll likely
only need:

```sh
cat > integration_config.json <<EOF
{ "apps_domain": "10.244.0.34.xip.io" }
EOF
```

### Running

To run the CF Acceptance tests, you need to be logged in and targeting an empty space and org (we suggest the `cats-space` inside the `cats-org`).

```sh
gcf api api.10.244.0.34.xip.io
gcf auth admin admin
gcf create-org cats-org
gcf target -o cats-org
gcf create-space cats-space
gcf target -o cats-org -s cats-space
```

```sh
export CONFIG=$PWD/integration_config.json
./bin/test [ginkgo arguments ...]
```

The `test` script will pass any given arguments to `ginkgo`, so this is where
you pass `-focus=`, `-nodes=`, etc.

#### Running in parallel

To run the tests in parallel, pass `-nodes=X`, where X is how many examples to
run at once.

```sh
./bin/test -nodes=10
```

Be careful with this number, as it's effectively "how many apps to push at
once", as every example probably pushes an app. For a [BOSH Lite][bosh-lite]
deployment, you may want to just set this to `10`. For a larger AWS deployment,
`100` may be just fine.

#### Seeing command-line output

If you want to see the output of all of the commands it shells out to, set
`CF_VERBOSE_OUTPUT` to `true`.

```sh
export CF_VERBOSE_OUTPUT=true
./bin/test
```

#### Capturing CF cli output

If `CF_TRACE_BASENAME` is set, then `CF_TRACE` will be set to `${CF_TRACE_BASENAME}${Ginko Node Id}.txt`
for each invocation of `gcf`.

##### Example:

```sh
export CF_TRACE_BASENAME=cf_trace_
./bin/test -nodes=10
```
The following files may be created:

```sh
cf_trace_1.txt
cf_trace_2.txt
cf_trace_3.txt
cf_trace_4.txt
cf_trace_5.txt
cf_trace_6.txt
cf_trace_7.txt
cf_trace_8.txt
cf_trace_9.txt
cf_trace_10.txt
```
If a test fails, look for the node id is the test output:

```sh
=== RUN TestLifecycle

Running Suite: Application Lifecycle
====================================
Random Seed: 1389376383
Parallel test node 2/10. Assigned 14 of 137 specs.
```

The `gcf` trace output for the tests in these specs will be in in `cf_trace_2.txt`

[cf-release]: https://github.com/cloudfoundry/cf-release
[ginkgo]: https://github.com/onsi/ginkgo
[bosh-lite]: https://github.com/cloudfoundry/bosh-lite
[cli]: https://github.com/cloudfoundry/cli
[go]: http://golang.org
