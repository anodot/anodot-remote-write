#!/usr/bin/env bash

root_folder=$(git rev-parse --show-toplevel)
files=$(gofmt -l "${root_folder}")
if [[ -n "${files}" ]]; then
    echo "Go files must be formatted with gofmt. Please run:"
    echo "gofmt -w ${root_folder}"

    printf "Not formatted files:\n"
    echo "${files}"
    exit 1
fi