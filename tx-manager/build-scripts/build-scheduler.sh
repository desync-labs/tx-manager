#!/bin/sh

# Define the version number as a constant
VERSION="0.0.1-dev"
APP_NAME="com.fathom.tx-manager.scheduler"
ORG_NAME="intothefathom"

echo 'Changing directory to node apps'

cd ../scheduler

echo `pwd`
echo "Creating image... ${ORG_NAME}/${APP_NAME}:${VERSION}"

docker build -t $ORG_NAME/$APP_NAME:$VERSION .
echo 'scheduler image created.'

# Output the version number to be used in the GitHub Action
echo "SCHEDULER_VERSION=$VERSION" >> $GITHUB_ENV
echo "SCHEDULER_NAME=$APP_NAME" >> $GITHUB_ENV
echo "ORG_NAME=$ORG_NAME" >> $GITHUB_ENV

cd ..
