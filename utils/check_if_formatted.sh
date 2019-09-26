#!/usr/bin/env bash

root_folder=$(git rev-parse --show-toplevel)
source_files=()
while IFS=  read -r -d $'\0'; do
    source_files+=("$REPLY")
done < <(find "${root_folder}" -path '*/vendor/*' -prune -o -name '*.go' -type f -print0)

files=$(gofmt -l ${source_files[*]})
if [[ -n "${files}" ]]; then
    echo "Go files must be formatted with gofmt. Please run:"
    echo -e "gofmt -w ${root_folder}\n"

    echo -e "Not formatted files:"
    echo "${files}"
    exit 1
fi