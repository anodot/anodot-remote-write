#!/usr/bin/env bash

set -u
current_dir="$(dirname "$0")"

declare -a supported_k8s_version=("1.13.0" "1.14.0" "1.15.0" "1.16.0" "1.17.0")

for i in "${supported_k8s_version[@]}"
do
   echo "Validating against '${i}' kubernetes version"
   helm kubeval ${current_dir}/../deployment/helm/* -v "$i"
done