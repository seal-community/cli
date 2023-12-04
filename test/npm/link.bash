#!/bin/bash
set -e
OUTPUT_DIR=$1/link_test
OUTPUT_FILE_NAME=$(npm -v)_link_dep.json

! rm -r $OUTPUT_DIR
mkdir -p $OUTPUT_DIR
cd $OUTPUT_DIR

# install
npm init -y
npm i -g semver-regex@0.1.1
npm link semver-regex

npm ll --all --json > $OUTPUT_FILE_NAME || true  # ignore errors
