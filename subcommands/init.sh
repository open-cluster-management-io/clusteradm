# Copyright Contributors to the Open Cluster Management project
# Command for starting the open-cluster-management control plane
function init_description {
  echo "Starting the open-cluster-management control plane"
}

function init_usage {
  errEcho "usage: $(basename ${0}) init [CONTEXT]"
  errEcho
  errEcho "    $(init_description)"
  errEcho
  errEcho "    CONTEXT is the name of a kubeconfig context"
  errEcho
  abort
}

function init {
  local context=$1
  if [[ -z $context ]]
  then
    kubectl cluster-info 
  else
    kubectl --context=$context cluster-info 
  fi
}
