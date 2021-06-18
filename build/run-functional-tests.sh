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

function init_hub() {
   echo "init_hub 1st parameter: "$1 >&2
   local _CMDINITRESULT=`clusteradm init $1`
   if [ $? != 0 ]
   then
      ERROR_REPORT=$ERROR_REPORT+"clusteradm init failed\n"
   fi
   echo $_CMDINITRESULT
}

function join_hub() {
   echo "join_hub 1st parameter: "$1 >&2
   echo "join_hub 2nd parameter: "$2 >&2
   local _CMDJOIN=`echo "$1" | cut -d ':' -f2,3,4 | cut -d '<' -f1`
   _CMDJOIN="$_CMDJOIN $2"
   local _CMDJOINRESULT=`$_CMDJOIN`
   if [ $? != 0 ]
   then
      ERROR_REPORT=$ERROR_REPORT+"clusteradm join failed\n"
   fi
   echo $_CMDJOINRESULT
}

function accept_cluster() {
   echo "accept_cluster 1st parameter: "$1 >&2
   local _CMDACCEPT=`echo "$1" | cut -d ':' -f2`
   _CMDACCEPT="$_CMDACCEPT"
   local _CMDACCEPTRESULT=`$_CMDACCEPT`
   if [ $? != 0 ]
   then
      ERROR_REPORT=$ERROR_REPORT+"clusteradm accept failed\n"
   fi
   echo $_CMDACCEPTRESULT
}

function gettoken() {
   local _CMDINITRESULT=`clusteradm get token`
   if [ $? != 0 ]
   then
      ERROR_REPORT=$ERROR_REPORT+"clusteradm get token failed\n"
   fi
   echo $_CMDINITRESULT
}

function joinscenario() {
   echo "joinscenario 1st parameter: "$1 >&2
   echo "joinscenario 2nd parameter: "$2 >&2
   echo "init cluster" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-hub 
   CMDINITRESULT=$(init_hub $2)
   echo "init command result: "$CMDINITRESULT >&2

   echo "join hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-$1
   CMDJOINRESULT=$(join_hub "${CMDINITRESULT}" $1)
   echo "join command result: "$CMDJOINRESULT >&2

   echo "Sleep 4 min to stabilize" >&2
   # we need to wait 2 min but once we will have watch status monitor
   # we will not need to sleep anymore
   sleep 240

   kubectl config use-context kind-${CLUSTER_NAME}-hub
   CMDACCEPTRESULT=$(accept_cluster "${CMDJOINRESULT}")
   echo $CMDACCEPTRESULT | grep approved
   if [ $? != 0 ]
   then
      echo "accept command result: "$CMDACCEPTRESULT >&2
      ERROR_REPORT=$ERROR_REPORT+"no CSR get approved\n"
   else
      echo "accept command result: "$CMDACCEPTRESULT >&2
   fi
}

function gettokenscenario() {
   echo "gettokenscenario 1st parameter: "$1 >&2
   echo "get token from hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-hub 
   CMGETTOKENRESULT=$(gettoken)
   echo "get token command result: "$CMGETTOKENRESULT >&2

   echo "join hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-$1
   CMDJOINRESULT=$(join_hub "${CMGETTOKENRESULT}" $1)
   echo "join command result: "$CMDJOINRESULT >&1

   echo "Sleep 4 min to stabilize" >&2
   # we need to wait 2 min but once we will have watch status monitor
   # we will not need to sleep anymore
   sleep 240

   kubectl config use-context kind-${CLUSTER_NAME}-hub
   CMDACCEPTRESULT=$(accept_cluster "${CMDJOINRESULT}")
   echo $CMDACCEPTRESULT | grep approved
   if [ $? != 0 ]
   then
      echo "accept command result: "$CMDACCEPTRESULT >&2
      ERROR_REPORT=$ERROR_REPORT+"no CSR get approved\n"
   else
      echo "accept command result: "$CMDACCEPTRESULT >&2
   fi
}

echo "With bootstrap token"
echo "--------------------"
export KUBECONFIG=$TEST_DIR/tmp/config.yaml
kind create cluster --name ${CLUSTER_NAME}-hub --config $TEST_DIR/kind-config/kind119-hub.yaml
kind create cluster --name ${CLUSTER_NAME}-c1
#Wait for cluster to setup
echo "Sleep 10 sec"
sleep 10

echo "Test clusteradm version"
clusteradm version
if [ $? != 0 ]
then
   ERROR_REPORT=$ERROR_REPORT+"clusteradm version failed\n"
fi

echo "Joining with init and bootstrap token"
echo "-------------------------------------"
joinscenario c1 --use-bootstrap-token 
kind delete cluster --name ${CLUSTER_NAME}-c1
kind create cluster --name ${CLUSTER_NAME}-c2
echo "Joining with get token and bootstrap token"
echo "------------------------------------------"
gettokenscenario c2

kind delete cluster --name ${CLUSTER_NAME}-hub
kind delete cluster --name ${CLUSTER_NAME}-c2

echo "With Service account"
echo "--------------------"
export KUBECONFIG=$TEST_DIR/tmp/config.yaml
kind create cluster --name ${CLUSTER_NAME}-hub --config $TEST_DIR/kind-config/kind119-hub.yaml
kind create cluster --name ${CLUSTER_NAME}-c1
#Wait for cluster to setup
sleep 10

echo "Joining with init and service account"
echo "-------------------------------------"
joinscenario c1
kind delete cluster --name ${CLUSTER_NAME}-c1
kind create cluster --name ${CLUSTER_NAME}-c2
echo "Joining with get token and service account"
echo "------------------------------------------"
gettokenscenario c2

kind delete cluster --name ${CLUSTER_NAME}-hub
kind delete cluster --name ${CLUSTER_NAME}-c2

if [ -z "$ERROR_REPORT" ]
then
    echo "Success"
else
    echo -e "\n\nErrors\n======\n"$ERROR_REPORT
    exit 1
fi
