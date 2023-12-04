#!/bin/bash
set -e
OUTPUT_DIR=$1/alias_test
OUTPUT_FILE_NAME=$(npm -v)_alias_dep.json

! rm -r $OUTPUT_DIR
mkdir -p $OUTPUT_DIR
cd $OUTPUT_DIR

# install
npm init -y
npm i is-extendable@npm:semver-regex@0.1.1
npm i extend-shallow@2.0.1 # has 1 dep: is-extendable ^0.1.0

npm ll --all --json > $OUTPUT_FILE_NAME
