#!/bin/sh

cf6 push v6 --no-start -p ~/workspace/cf-acceptance-tests/assets/dora
cf6 delete v6 -f -r -v > /tmp/v6-delete-app.txt


cf push v7 --no-start -p ~/workspace/cf-acceptance-tests/assets/dora
cf delete v7 -f -r -v > /tmp/v7-delete-app.txt
