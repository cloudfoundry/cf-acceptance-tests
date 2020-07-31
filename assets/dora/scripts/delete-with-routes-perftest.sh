#!/bin/sh

cf push v7 --no-start -p ~/workspace/cf-acceptance-tests/assets/dora
for i in $(seq 0 500); do cf map-route v7 jelly-ferret.lite.cli.fun --hostname v7-$i ; done
time cf delete v7 -f -r > /tmp/v7-delete-perf-test-time.txt

cf push v6 --no-start -p ~/workspace/cf-acceptance-tests/assets/dora
for i in $(seq 0 500); do cf map-route v6 jelly-ferret.lite.cli.fun --hostname v6-$i ; done
time cf delete v6 -f -r > /tmp/v6-delete-perf-test-time.txt

cf push v7 --no-start -p ~/workspace/cf-acceptance-tests/assets/dora
for i in $(seq 0 5); do cf map-route v7 jelly-ferret.lite.cli.fun --hostname v7-$i ; done
bosh stop cc-worker
sleep 10
cf delete v7 -f -r -v > /tmp/v7-delete-perf-test-time.txt
