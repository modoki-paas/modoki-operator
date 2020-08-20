#! /bin/sh
set -eu

# Build CDK
cd /cdk8s-template && npm run build

# Start controller
/manager $@