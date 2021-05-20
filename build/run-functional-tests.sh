#!/bin/bash
# Copyright Contributors to the Open Cluster Management project

# set -x
set -e
TEST_DIR=test/functional
TEST_RESULT_DIR=$TEST_DIR/tmp
ERROR_REPORT=""
CLUSTER_NAME=$PROJECT_NAME-functional-test
export KUBECONFIG=$TEST_DIR/tmp/kind.yaml

rm -rf $TEST_RESULT_DIR
mkdir -p $TEST_RESULT_DIR

kind create cluster --name ${CLUSTER_NAME}
#Wait for cluster to setup
sleep 10

echo "Test clusteradm get secret"
clusteradm get secret -n default
if [ $? != 0 ]
then
   ERROR_REPORT=$ERROR_REPORT+"clusteradm get secret -n default failed\n"
fi

if [ -z "$ERROR_REPORT" ]
then
    echo "Success"
else
    echo -e "\n\nErrors\n======\n"$ERROR_REPORT
    exit 1
fi

kind delete cluster --name $CLUSTER_NAME
