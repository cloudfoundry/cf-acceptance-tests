Nora
====

Nora .NET api app for testing

A sister app of Dora


Install
=======

To install you will have to get cf 6.10+ and run the following commands:

```sh
cf add-plugin-repo CF-Community http://plugins.cloudfoundry.org/
cf install-plugin Diego-Beta -r CF-Community
```

Run the following command to deploy nora:

```sh
./make_a_nora <app_name> <stack_name>
```

Requirements
=======
Nora requires at least 512mb of memory to run on CloudFoundry.
