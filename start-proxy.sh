#!/bin/bash

# Local development script for kubevirt-apiserver-proxy

export APP_ENV=dev

KUBE_API_SERVER=$(oc config view --minify -o jsonpath='{.clusters[0].cluster.server}' | sed 's|https://||')

if [ -z "$KUBE_API_SERVER" ]; then
  echo "Error: Failed to get API server from oc config. Make sure you're logged in."
  exit 1
fi

export KUBE_API_SERVER

echo "Using API Server: $KUBE_API_SERVER"

go build
if [ $? -ne 0 ]; then
  echo "Error: Build failed"
  exit 1
fi

./kubevirt-apiserver-proxy
