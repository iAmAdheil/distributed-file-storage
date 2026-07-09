#!/usr/bin/env bash
#
# clean_networks.sh — remove the per-node storage roots (":<port>_network"
# folders) that the file server creates in the working directory.
#
# Edit PORTS to match the nodes you spin up.

set -euo pipefail

# Ports whose ":<port>_network" folder should be removed.
PORTS=(3000 4000 5000 6000 7000)

# Run relative to the repo root regardless of where the script is invoked from.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

removed=0
for port in "${PORTS[@]}"; do
	dir="${ROOT}/:${port}_network"
	if [[ -d "$dir" ]]; then
		rm -rf "$dir"
		echo "removed ${dir}"
		removed=$((removed + 1))
	else
		echo "skip   ${dir} (not found)"
	fi
done

echo "done: ${removed} folder(s) removed"
