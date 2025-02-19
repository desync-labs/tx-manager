#!/bin/sh

# Define the version number as a constant
VERSION="0.0.2-dev"
APP_NAME="com.fathom.tx-manager.key-manager"
ORG_NAME="intothefathom"

echo 'Changing directory to node apps'

cd ../key-manager

echo `pwd`
echo "Creating image... ${APP_NAME}:${VERSION}"

docker build -t $ORG_NAME/$APP_NAME:$VERSION .
echo 'key manager image created.'

# Output the version number to be used in the GitHub Action
echo "KEY_MANAGER_VERSION=$VERSION" >> $GITHUB_ENV
echo "KEY_MANAGER_NAME=$APP_NAME" >> $GITHUB_ENV
echo "ORG_NAME=$ORG_NAME" >> $GITHUB_ENV


cd ..


