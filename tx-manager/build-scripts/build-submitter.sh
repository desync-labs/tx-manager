#!/bin/sh

# Define the version number as a constant
VERSION="0.0.1.1-dev"
NAME="com.fathom.tx-manager.submitter"
ORG="intothefathom"

echo 'Changing directory to node apps'

cd ../../submitter

echo `pwd`
echo "Creating image... ${APP_NAME}:${VERSION}"

docker build -t $ORG_NAME/$APP_NAME:$VERSION .
echo 'submitter image created.'

# Output the version number to be used in the GitHub Action
echo "SUBMITTER_VERSION=$VERSION" >> $GITHUB_ENV
echo "SUBMITTER_NAME=$NAME" >> $GITHUB_ENV
echo "ORG_NAME=$ORG" >> $GITHUB_ENV

cd ..


