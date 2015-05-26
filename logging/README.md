# Operator logging tests

Tests in this package are only intended to be run on machines that are accessible by your deployment.

## Purpose
This test exercises the syslog drain forwarding functionality. A TCP listener is
spun up on the running machine, an app is deployed to the target Cloud Foundry
and bound to that listener (as a syslog drain) and the drain is checked for log
messages.
