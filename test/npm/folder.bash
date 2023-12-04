#!/bin/bash
set -e
OUTPUT_DIR=$1/folder_test
OUTPUT_FILE_NAME=$(npm -v)_folder_dep.json

! rm -r $OUTPUT_DIR
mkdir -p $OUTPUT_DIR
cd $OUTPUT_DIR

# create dependency folder
npm pack is-number@6.0.0
tar -xzf is-number-6.0.0.tgz -C /tmp
rm -f is-number-6.0.0.tgz
# install
npm init -y
npm i /tmp/package --install-links

npm ll --all --json > $OUTPUT_FILE_NAME || true  # ignore errors
