#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

main() {
  pushd "$ROOT" &> /dev/null

  while getopts "h" opt; do
    case $opt in
      h) usage && exit 0;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))

  set -e

  echo "Installing all command line utilities found in './cmd'"
  for folder in `ls ./cmd`; do
    go install "./cmd/$folder"
  done
}

usage_error() {
  message="$1"
  exit_code="$2"

  echo "ERROR: $message"
  echo ""
  usage
  exit ${exit_code:-1}
}

usage() {
  echo "usage: install_all.sh <option>"
  echo ""
  echo "Install all Golang binary tools found in this repository. This"
  echo "script simply call 'go install ./cmd/<directory>' on all directories"
  echo "found inside './cmd' folder."
  echo ""
  echo "Options"
  echo "    -h          Display help about this script"
}

main "$@"