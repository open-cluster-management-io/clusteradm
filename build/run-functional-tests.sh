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

export KUBECONFIG=$TEST_DIR/tmp/config.yaml
kind create cluster --name ${CLUSTER_NAME}-hub --config $TEST_DIR/kind-config/kind119-hub.yaml
kind create cluster --name ${CLUSTER_NAME}-spoke
#Wait for cluster to setup
sleep 10

echo "Test clusteradm version"
clusteradm version
if [ $? != 0 ]
then
   ERROR_REPORT=$ERROR_REPORT+"clusteradm version failed\n"
fi

kubectl config use-context kind-${CLUSTER_NAME}-hub 
CMDINITRESULT=`clusteradm init --use-bootstrap-token`
if [ $? != 0 ]
then
   echo "init command result: "$CMDINITRESULT
   ERROR_REPORT=$ERROR_REPORT+"clusteradm init failed\n"
else
   echo "init command result: "$CMDINITRESULT
   echo $CMDINITRESULT
fi

CMDJOIN=`echo $CMDINITRESULT | cut -d ':' -f2,3,4 | cut -d '<' -f1`
CMDJOIN="$CMDJOIN c1"
echo "Join command: "$CMDJOIN
kubectl config use-context kind-${CLUSTER_NAME}-spoke
CMDJOINRESULT=`$CMDJOIN`
if [ $? != 0 ]
then
   echo "join command result: " $CMDJOINRESULT
   ERROR_REPORT=$ERROR_REPORT+"clusteradm join failed\n"
else
   echo "join command result: " $CMDJOINRESULT
fi

echo "Sleep 4 min to stabilize"
# we need to wait 2 min but once we will have watch status monitor
# we will not need to sleep anymore
sleep 240

CMDACCEPT=`echo $CMDJOINRESULT | cut -d ':' -f2`
CMDACCEPT="$CMDACCEPT c1"
echo "accept command: "$CMDACCEPT
kubectl config use-context kind-${CLUSTER_NAME}-hub
CMDACCEPTRESULT=`$CMDACCEPT`
if [ $? != 0 ]
then
   echo "accept command result: "$CMDACCEPTRESULT
   ERROR_REPORT=$ERROR_REPORT+"clusteradm accept failed\n"
else
   echo "accept command result: "$CMDACCEPTRESULT
fi

echo $CMDACCEPTRESULT | grep approved
if [ $? != 0 ]
then
   echo "accept command result: "$CMDACCEPTRESULT
   ERROR_REPORT=$ERROR_REPORT+"no CSR get approved\n"
else
   echo "accept command result: "$CMDACCEPTRESULT
fi

if [ -z "$ERROR_REPORT" ]
then
    echo "Success"
else
    echo -e "\n\nErrors\n======\n"$ERROR_REPORT
    exit 1
fi

kubectl config use-context kind-${CLUSTER_NAME}-hub 
CMDINITRESULT=`clusteradm get token`
if [ $? != 0 ]
then
   echo "get token command result: "$CMDINITRESULT
   ERROR_REPORT=$ERROR_REPORT+"clusteradm get token failed\n"
else
   echo "get token command result: "$CMDINITRESULT
   echo $CMDINITRESULT
fi

kind delete cluster --name $CLUSTER_NAME-hub
kind delete cluster --name $CLUSTER_NAME-spoke
