#!/bin/sh

# Define the version number as a constant
VERSION="0.0.2-dev"
APP_NAME="com.fathom.tx-manager.submitter"
ORG_NAME="intothefathom"

echo 'Changing directory to node apps'

cd ../submitter

echo `pwd`
echo "Creating image... ${ORG_NAME}/${APP_NAME}:${VERSION}"

docker build -t $ORG_NAME/$APP_NAME:$VERSION .
echo 'submitter image created.'

# Output the version number to be used in the GitHub Action
echo "SUBMITTER_VERSION=$VERSION" >> $GITHUB_ENV
echo "SUBMITTER_NAME=$APP_NAME" >> $GITHUB_ENV
echo "ORG_NAME=$ORG_NAME" >> $GITHUB_ENV

cd ..


