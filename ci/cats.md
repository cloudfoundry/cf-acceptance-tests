# cats

## Purpose

The main purpose of this pipeline is to validate that any changes made to cf-acceptance-tests are tested for compatibility with the latest release of [cf-deployment](https://github.com/cloudfoundry/cf-deployment).

Additionally, the pipeline includes manually triggered jobs to cut releases of cf-acceptance-tests.

## Validation Strategy

### Unit Testing

As a blocking step before running acceptance test suites, we run a separate suite of unit tests to validate the configuration interface and cf CLI integration. Currently, the suites are located in the `helpers/config` and `helpers/cli_version_check` directories.

### Acceptance Testing

We validate every test suite but windows, using the following config to enable those test suites:

```
  "include_apps": true,
  "include_backend_compatibility": true,
  "include_container_networking": true,
  "include_detect": true,
  "include_docker": true,
  "include_internet_dependent": true,
  "include_private_docker_registry": true,
  "include_route_services": true,
  "include_routing": true,
  "include_security_groups": true,
  "include_service_instance_sharing": true,
  "include_services": true,
  "include_ssh": true,
  "include_sso": true,
  "include_tasks": true,
  "include_tcp_routing": true,
  "include_v3": true,
  "include_windows": false,
  "include_zipkin": true
```

## Infrastructure

This pipeline claims infrastructure provisioned by other pipelines through concourse resource-pool resources. For the management of these resources, `acquire-pool` and `release-pool` jobs are present for each type of infrastructure used. If a job fails and you are done investigating the environment, remember to clean up the environment manually release the pool lock.

### CF

We deploy the latest [cf-deployment](https://github.com/cloudfoundry/cf-deployment) release with an isolation segment enabled. The Bosh director is provisioned by a separate infrastructure pipeline that uses the bosh-bootloader utility.

The deployment is managed by the `deploy-cf` and `cleanup-cats` jobs which deploy and clean up cf-deployment respectively.

## Release Management

Release management is a manual process and consists of the following steps:

1. Manually generate release notes based on the current diff between the `release-candidate` and `main` branches
1. Choose whether the next release is going to be a major, minor or patch
   release.
1. Run the corresponding `ship-it-*` job based on the release type identified in the previous step to promote the changes on the `release-candidate` branch and create a new release tag on the `main` branch.
1. Use the newly created release tag and your release notes to create a new github release

See the [Releasing wiki page](https://github.com/cloudfoundry/cf-acceptance-tests/wiki/Releasing) for more details.

Note that at the time of writing, when a new version of cf-deployment is released, a job in the cf-acceptance-tests pipeline also creates a new branch at the cf-deployment release tag that freezes the versions of cf-deployment and cf-acceptance-tests.

## Pipeline management

This pipeline is managed by the `ci/pipeline.yml` file. To make changes to the pipeline, update the file directly and either run the `ci/configure` script to apply the changes (if you've already set up your fly cli to use the `ard` target) or manually run the fly cli `set-pipeline` command.
