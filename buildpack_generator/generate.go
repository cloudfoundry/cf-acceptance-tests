package buildpack_generator

import (
	"fmt"
	"os"
	"path"
)

func GenerateBuildpack(destination, matchingFilename string) error {
	err := os.Mkdir(path.Join(destination, "bin"), 0755)
	if err != nil {
		return err
	}

	err = writeBuildpackCompile(destination)
	if err != nil {
		return err
	}

	err = writeBuildpackDetect(destination, matchingFilename)
	if err != nil {
		return err
	}

	return writeBuildpackRelease(destination)
}

func writeBuildpackCompile(buildpackPath string) error {
	compile, err := os.OpenFile(path.Join(buildpackPath, "bin", "compile"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = compile.WriteString(
		`#!/usr/bin/env bash

sleep 1

echo "Staging with Simple Buildpack"
`)
	if err != nil {
		return err
	}

	return compile.Close()
}

func writeBuildpackDetect(buildpackPath, matchingFile string) error {
	detect, err := os.OpenFile(path.Join(buildpackPath, "bin", "detect"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = detect.WriteString(
		fmt.Sprintf(`#!/bin/bash

if [ -f "${1}/%s" ]; then
  echo Simple
else
  echo no
  exit 1
fi
`, matchingFile),
	)
	if err != nil {
		return err
	}

	return detect.Close()
}

func writeBuildpackRelease(buildpackPath string) error {
	release, err := os.OpenFile(path.Join(buildpackPath, "bin", "release"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = release.WriteString(
		`#!/usr/bin/env bash

cat <<EOF
---
config_vars:
  PATH: bin:/usr/local/bin:/usr/bin:/bin
  FROM_BUILD_PACK: "yes"
default_process_types:
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; } | nc -l \$PORT; done
EOF
`)
	if err != nil {
		return err
	}

	return release.Close()
}
