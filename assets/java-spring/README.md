# Spring Boot Trivial App

This contains the built JAR, and the source code for a trivial SpringBoot app.

The app has one feature which is to return `ok`.

The source code is included so that future consumers of this app can make modifications.

A pre-built JAR is included so that no Java toolchain is needed when running cats.

## Building

The application JAR can be rebuilt by running `mvn package` provided a JDK is
available and Maven version  `3.5+` is in installed.

## Context

The app is build using SpringBoot 2.7.7, following the instructions found
[here](https://docs.spring.io/spring-boot/docs/2.7.7/reference/html/getting-started.html#getting-started.first-application).
