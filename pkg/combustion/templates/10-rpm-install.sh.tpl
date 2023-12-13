#!/bin/bash
set -euo pipefail

#  Template Fields
#  RPMs - A string that contains all of the RPMs present in the user created config directory, separated by spaces.

{{ if ne .RepoName "" -}}
zypper ar file://{{.RepoPath}} {{.RepoName}}
zypper --no-gpg-checks install -r {{.RepoName}} -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
zypper rr {{.RepoName}}
{{ else }}
zypper --no-gpg-checks install -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
{{ end }}