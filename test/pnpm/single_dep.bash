#!/bin/bash
set -e
OUTPUT_DIR=$1/single_dep_test
OUTPUT_FILE_NAME=$(npm -v)_single_dep.json

! rm -r $OUTPUT_DIR
mkdir -p $OUTPUT_DIR
cd $OUTPUT_DIR

# install
npm init -y
pnpm add semver-regex@1.0.0
pnpm install
pnpm ll --depth Infinity --json > $OUTPUT_FILE_NAME