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

Building Nora
=============

#### Requirements:

* Microsoft Windows OS

* [Msbuild.exe](https://docs.microsoft.com/en-us/visualstudio/msbuild/msbuild)

#### Build

* Make sure you have `msbuild.exe` on your `$PATH`.

* Make your code changes in the `Nora/` directory

* Run `./make.bat`

* This is will build the app in the `Nora/` directory and you'll see it has a new `bin/` directory.

#### An easier way to build nora

You can also use the dotnet-framework docker image which has `msbuild` preinstalled to easily build nora.

* Make your code changes in the `Nora/` directory

* Pull the docker image: `docker pull mcr.microsoft.com/dotnet/framework/sdk`

* Run a container with `nora/` mounted: `docker run --rm -it -v <path/to/nora>\nora:C:\nora mcr.microsoft.com/dotnet/framework/sdk powershell`

* Inside the container: `cd nora`, and `./make.bat`. This should build the app inside `Nora/`

* Exit the container using `exit`.
