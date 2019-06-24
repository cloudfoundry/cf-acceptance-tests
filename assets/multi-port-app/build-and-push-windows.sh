#!/bin/sh

echo "Building the binary."
GOOS=windows go build

echo "Pushing the app to the windows stack."
cf push multi -s windows -b binary_buildpack -c "./multi-port-app.exe --ports=8080,8081"

echo "Capturing the app guid."
export MULTI_APP_GUID=$(cf app multi --guid)

echo "Configuring CF with the ports the app is listening on."
cf curl /v2/apps/$MULTI_APP_GUID -X PUT -d '{"ports": [8080, 8081]}'

echo "Capturing the route guid."
export MULTI_ROUTE_GUID=$(cf curl /v2/routes?q=host:multi | jq -r .resources[0].metadata.guid)

echo "Updating the route mapping for the app."
cf curl /v2/route_mappings -X POST -d '{"app_guid": "'"$MULTI_APP_GUID"'", "route_guid": "'"$MULTI_ROUTE_GUID"'", "app_port": 8081}'
