#!/bin/bash

docker run -v $PWD:/home/cats -w /home/cats cloudfoundry/cf-deployment-concourse-tasks go build -o app
