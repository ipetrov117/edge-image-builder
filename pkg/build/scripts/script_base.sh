#!/bin/bash
set -euo pipefail

# Redirect output to the console
exec > >(exec tee -a /dev/tty0) 2>&1

cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1

REPO_LOC=/dev/shm/combustion/config/repo
zypper ar file://$REPO_LOC airgap-repo
zypper --no-gpg-checks install -y --force-resolution --auto-agree-with-licenses $(cat $REPO_LOC/package-list.txt)
zypper rr airgap-repo