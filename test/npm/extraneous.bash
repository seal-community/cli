#!/bin/bash
set -e
OUTPUT_DIR=$1/extraneous_test
OUTPUT_FILE_NAME=$(npm -v)_extraneous.json

! rm -r $OUTPUT_DIR
mkdir -p $OUTPUT_DIR
cd $OUTPUT_DIR

# install
npm init -y
npm i semver-regex@1.0.0 --no-save
npm ll --all --json > $OUTPUT_FILE_NAME
