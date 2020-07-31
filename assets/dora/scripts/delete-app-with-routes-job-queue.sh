#!/bin/sh

cf push v7 --no-start -p ~/workspace/cf-acceptance-tests/assets/dora
for i in $(seq 0 5); do cf map-route v7 jelly-ferret.lite.cli.fun --hostname v7-$i ; done
cf delete v7 -f -r -v > /tmp/v7-delete-app-and-5routes-job-queue.txt

