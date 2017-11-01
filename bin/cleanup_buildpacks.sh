#!/usr/bin/env bash

# script to cleanup CATS buildpacks
# make sure you are targeting the right environment (both cf and uaac) and the CATS are not running.

set -e

COUNTER=1
while [  $COUNTER -lt 6 ]; do
  echo "Counter = $COUNTER"

  echo "Cleaning up buildpacks..."
  cf buildpacks | grep buildpack.zip | cut -f1 -d ' ' | while read -r buildpack ; do
      echo "About to delete buildpack: $buildpack"
      cf delete-buildpack $buildpack -f
  done

  let COUNTER=COUNTER+1
done


