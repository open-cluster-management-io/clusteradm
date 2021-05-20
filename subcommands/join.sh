# Copyright Contributors to the Open Cluster Management project
# Command for managed cluster to join hub cluster
function join_description {
  echo "Initializes a klusterlet agent on the managed cluster and joins it to the hub cluster."
}

function join_usage {
  errEcho "usage: $(basename ${0}) join [CONTEXT]"
  errEcho
  errEcho "    $(join_description)"
  errEcho
  errEcho "    CONTEXT is the name of a kubeconfig context"
  errEcho
  abort
}

function join {
  local context=$1
  if [[ -z $context ]]
  then
    kubectl cluster-info 
  else
    kubectl --context=$context cluster-info 
  fi
}
