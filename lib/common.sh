# Copyright Contributors to the Open Cluster Management project

VERBOSITY=0
COMMAND_VERBOSITY=2
OUTPUT_VERBOSITY=3

VERIFIED_CONTEXTS=()

function abort {
  kill -s TERM $TOP_PID
}

function errEcho {
  echo >&2 "$@"
}

function verbose {
  local verbosity=$1
  shift
  if [[ $verbosity -le $VERBOSITY ]]
  then
    errEcho "$@"
  fi
}

function logCommand {
  verbose $COMMAND_VERBOSITY "command: $@"
}

function logOutput {
  verbose $OUTPUT_VERBOSITY "output:"
  verbose $OUTPUT_VERBOSITY "$@"
}

function error {
  echo >&2 "error: $@"
}

function fatal {
  error "$@"
  abort
}

# Use to silence output when not using "return" value from a function
function ignoreOutput {
  "$@" > /dev/null
}

# Use to log regular commands
# - command is logged to stderr if VERBOSITY is COMMAND_VERBOSITY or higher
# - command output is logged to stderr if VERBOSITY is OUTPUT_VERBOSITY or higher
# - stderr is not suppressed; failure exits the script
function cmd {
  logCommand "$@"
  OUTPUT=$("$@")
  logOutput "$OUTPUT"
}

# Use to log regular commands that can fail
# - command is logged to stderr if VERBOSITY is COMMAND_VERBOSITY or higher
# - command output is logged to stderr if VERBOSITY is OUTPUT_VERBOSITY or higher
# - stderr is suppressed and a failure will not exit the script
function cmdTry {
  set +e
  logCommand "$@"
  OUTPUT=$("$@" 2>&1)
  logOutput "$OUTPUT"
  set -e
}

# Use to log command substitutions or regular commands when output should be displayed to the user
# - command is logged to stderr if VERBOSITY is COMMAND_VERBOSITY or higher
# - command output is logged to stderr if VERBOSITY is OUTPUT_VERBOSITY or higher
# - stderr is not suppressed; failure exits the script
# - stdout is "returned"
function sub {
  logCommand "$@"
  OUTPUT=$("$@")
  logOutput "$OUTPUT"
  echo "$OUTPUT"
}

# Use to log command substitutions when only interested in exit code
# - command is logged to stderr if VERBOSITY is COMMAND_VERBOSITY or higher
# - command stdout and stderr are logged to stderr if VERBOSITY is OUTPUT_VERBOSITY or higher
# - stderr is suppressed and does not fail the script; "returns" exit code
function subRC {
  set +e
  logCommand $@
  OUTPUT=$("$@" 2>&1)
  RC=$?
  logOutput "$OUTPUT"
  set -e
  echo -n $RC
}

# Use to log command substitutions, ignoring errors
# - command is logged to stderr if VERBOSITY is COMMAND_VERBOSITY or higher
# - command output logged to stderr if VERBOSITY is OUTPUT_VERBOSITY or higher
# - stderr is not suppressed
# - stdout is "returned" if command exits successfully
function subIf {
  set +e
  logCommand $@
  OUTPUT=$("$@")
  RC=$?
  logOutput "$OUTPUT"
  if [[ $RC -eq 0 ]]
  then
    echo $OUTPUT
  fi
  set -e
}

# nd = new directory
function nd() {
  pushd $1 > /dev/null
}

# od = old directory
function od() {
  popd > /dev/null
}

function notImplemented {
  error "${FUNCNAME[1]} not implemented"
  abort
}

# Runs the given command in its own directory
function dirSensitiveCmd {
  local cmdDir=$(dirname "$1")
  nd "$cmdDir"
  $1
  od
}

function replaceString {
  local length
  local original="$1"
  local replacement="$2"
  local index="$3"
  length=${#replacement}
  echo "${original:0:${index}}${replacement}${original:$((index + length))}"
}

function getAge {
  local days hours minutes result
  local age=$1
  days=$((age / 86400))
  hours=$((age % 86400 / 3600))
  minutes=$((age % 3600 / 60))
  seconds=$((age % 60))
  result=""
  if [[ $days -gt 0 ]]
  then
    result="${days}d"
  fi
  if [[ $days -lt 7 && $hours -gt 0 ]]
  then
    result="${result}${hours}h"
  fi
  if [[ $days -eq 0 && $hours -lt 24 && $minutes -gt 0 ]]
  then
    result="${result}${minutes}m"
  fi
  if [[ $days -eq 0 && $hours -eq 0 && $minutes -lt 60 && $seconds -gt 0 ]]
  then
    result="${result}${seconds}s"
  fi
  echo $result
}

