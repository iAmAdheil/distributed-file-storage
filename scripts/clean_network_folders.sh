#!/usr/bin/env bash
#
# clean_networks.sh — remove the per-node storage roots (":<port>_network"
# folders) that the file server creates in the working directory.
#
# Edit PORTS to match the nodes you spin up.

set -euo pipefail

# Ports whose ":<port>_network" folder should be removed.
PORTS=(3000 4000 5000 6000 7000)

# Files to delete recursively from the repo root, matched by exact name.
# Add entries here, e.g. "test_db.db" or "debug.log".
JUNK_FILES=(
	"test_db.db"
	":3000.db"
	":4000.db"
	":7000.db"
)

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

# Empty the DBs directory but keep the directory itself.
dbs="${ROOT}/DBs"
if [[ -d "$dbs" ]]; then
	find "$dbs" -mindepth 1 -delete
	echo "cleared ${dbs}"
else
	echo "skip   ${dbs} (not found)"
fi

# Recursively delete any file whose name matches an entry in JUNK_FILES.
if ((${#JUNK_FILES[@]} > 0)); then
	name_args=()
	for name in "${JUNK_FILES[@]}"; do
		name_args+=(-o -name "$name")
	done
	name_args=("${name_args[@]:1}") # drop the leading -o

	deleted=0
	while IFS= read -r -d '' file; do
		rm -f "$file"
		echo "removed ${file}"
		deleted=$((deleted + 1))
	done < <(find "$ROOT" -path "${ROOT}/.git" -prune -o -type f \( "${name_args[@]}" \) -print0)

	echo "done: ${deleted} file(s) removed"
else
	echo "skip   file cleanup (JUNK_FILES is empty)"
fi
