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
   local _CMDINITRESULT=`clusteradm init $1 --image-registry=quay.io/open-cluster-management --tag=latest`
   if [ $? != 0 ]
   then
      ERROR_REPORT=$ERROR_REPORT+"clusteradm init failed\n"
   fi
   echo $_CMDINITRESULT
}

function join_hub() {
   echo "join_hub 1st parameter: "$1 >&2
   echo "join_hub 2nd parameter: "$2 >&2
   echo "join_hub 3nd parameter: "$3 >&2
   echo "join_hub 4nd parameter: "$4 >&2
   local _CMDJOIN=`echo "$1" | cut -d ':' -f2-4 | cut -d '<' -f1`
   _CMDJOIN="$_CMDJOIN $2 $3 $4"
   local _CMDJOINRESULT=`$_CMDJOIN --wait --force-internal-endpoint-lookup`
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
   local _CMDACCEPTRESULT=`$_CMDACCEPT --wait 240`
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

   echo "Wait 4 min maximum to stabilize" >&2
 
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

function joinscenario_with_timeout() {
   echo "joinscenario 1st parameter: "$1 >&2
   echo "joinscenario 2nd parameter: "$2 >&2
   echo "joinscenario 3nd parameter: "$3 >&2
   echo "init cluster" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-hub 
   CMDINITRESULT=$(init_hub)
   echo "init command result: "$CMDINITRESULT >&2

   echo "join hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-c1
   CMDJOINRESULT=$(join_hub "${CMDINITRESULT}" $1 $2 $3)
   echo "join command result: "$CMDJOINRESULT >&2

   echo "Wait 4 min maximum to stabilize" >&2
 
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
   echo "gettokenscenario 2nd parameter: "$2 >&2
   echo "get token from hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-hub 
   CMGETTOKENRESULT=$(gettoken $2)
   echo "get token command result: "$CMGETTOKENRESULT >&2

   echo "join hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-$1
   CMDJOINRESULT=$(join_hub "${CMGETTOKENRESULT}" $1)
   echo "join command result: "$CMDJOINRESULT >&1

   echo "Wait 4 min maximum to stabilize" >&2
 
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

   echo "delete token" >&2
   clusteradm delete token
      if [ $? != 0 ]
   then
      echo "accept command result: "$CMDACCEPTRESULT >&2
      ERROR_REPORT=$ERROR_REPORT+"no CSR get approved\n"
   fi

   echo "get token from hub" >&2
   kubectl config use-context kind-${CLUSTER_NAME}-hub 
   CMGETTOKENRESULT2=$(gettoken $2)
   if [ "$CMGETTOKENRESULT" == "$CMGETTOKENRESULT2" ]
   then
     ERROR_REPORT=$ERROR_REPORT+"new token identical as previous token after delete"
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
gettokenscenario c2 --use-bootstrap-token 

kind delete cluster --name ${CLUSTER_NAME}-hub
kind delete cluster --name ${CLUSTER_NAME}-c2

echo "With Service account"
echo "--------------------"
export KUBECONFIG=$TEST_DIR/tmp/config.yaml
kind create cluster --name ${CLUSTER_NAME}-hub --config $TEST_DIR/kind-config/kind119-hub.yaml
kind create cluster --name ${CLUSTER_NAME}-c1
#Wait for cluster to setup
echo "Sleep 10 sec"
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

echo "with timeout"
echo "-------------------------------------" 

export KUBECONFIG=$TEST_DIR/tmp/config.yaml
kind create cluster --name ${CLUSTER_NAME}-hub --config $TEST_DIR/kind-config/kind119-hub.yaml
kind create cluster --name ${CLUSTER_NAME}-c1
#Wait for cluster to setup
echo "Sleep 10 sec"
sleep 10

echo "Joining with timeout"
echo "-------------------------------------"
joinscenario_with_timeout c1 --timeout 400 
kind delete cluster --name ${CLUSTER_NAME}-hub
kind delete cluster --name ${CLUSTER_NAME}-c1


if [ -z "$ERROR_REPORT" ]
then
    echo "Success"
else
    echo -e "\n\nErrors\n======\n"$ERROR_REPORT
    exit 1
fi
