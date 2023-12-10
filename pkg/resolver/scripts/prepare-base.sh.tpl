#!/bin/bash
set -euo pipefail

INTERNAL_IMAGE_DIR={{.BaseImageDir}}
BASE_ISO_PATH={{.BaseISOPath}}

xorriso -osirrox on -indev $BASE_ISO_PATH extract / $INTERNAL_IMAGE_DIR/iso-root/
cd $INTERNAL_IMAGE_DIR/iso-root/
unsquashfs $INTERNAL_IMAGE_DIR/iso-root/SLE-Micro.raw.squashfs
cd $INTERNAL_IMAGE_DIR/iso-root/squashfs-root
virt-tar-out -a SLE-Micro.raw / - | gzip --best > $INTERNAL_IMAGE_DIR/{{.ArchiveName}}