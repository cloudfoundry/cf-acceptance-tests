#!/usr/bin/env bash

# script to cleanup CATS users, orgs and quotas
# make sure you are targeting the right environment (both cf and uaac) and the CATS are not running.

set -e

echo "Cleaning up orgs..."
cf orgs | grep 'CATS-' | while read -r org ; do
    echo "About to delete org: $org"
    cf delete-org $org -f
done

echo "Cleaning up quotas..."
cf quotas | grep 'CATS-' | cut -f1 -d ' ' | while read -r quota ; do
    echo "About to delete quota: $quota"
    cf delete-quota $quota -f
done

echo "Cleaning up buildpacks..."
cf buildpacks | grep buildpack.zip | cut -f1 -d ' ' | while read -r buildpack ; do
    echo "About to delete buildpack: $buildpack"
    cf delete-buildpack $buildpack -f
done

echo "Cleaning up service brokers..."
cf service-brokers | grep 'CATS-' | cut -f1 -d ' ' | while read -r service_broker ; do
    echo "About to delete service broker: $service_broker"
    cf delete-service-broker $service_broker -f
done

echo "Cleaning up domains..."
cf domains | grep 'CATS-' | cut -f1 -d ' ' | while read -r domain ; do
    echo "About to delete domain: $domain"
    cf delete-shared-domain $domain -f
done

echo "Cleaning up users..."
uaac users | grep username | grep 'CATS-USER-' | cut -f6 -d ' ' | while read -r user ; do
    echo "About to delete user: $user"
    cf delete-user $user -f
done
