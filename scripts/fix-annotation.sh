#!/usr/bin/env bash
set -euo pipefail

jsonpath='{.items[?(@.metadata.annotations.pv\.kubernetes\.io/provisioned-by=="ch.cloudscale.csi")].metadata.name}'
subcommand="${1:-""}"

pvs=$(
  kubectl get pv \
    -A \
    -o=jsonpath="${jsonpath}"
)

echo "Found the following persistent volumes (PVs):"
for pv in $pvs; do
  echo "${pv}"
done

if [ "$subcommand" != "migrate" ]
then
  echo "Run \"./${0} migrate\" to update the PVs."
  exit
fi

for pv in $pvs; do
  kubectl annotate pv --overwrite "${pv}" "pv.kubernetes.io/provisioned-by=csi.cloudscale.ch"
done
