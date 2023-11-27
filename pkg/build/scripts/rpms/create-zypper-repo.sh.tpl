#!/bin/bash
set -euo pipefail

# Clean up any previous variables
unset PKG_LIST
unset ISO_PATH
unset ADD_REPO
unset WORK_DIR

# Set the location of your SLE Micro SelfInstall ISO on the filesystem
ISO_NAME={{.ISOName}}

# Provide a list of packages that you want to install from SUSE repositories
PKG_LIST="{{.PKGList}}"

# # Provide a list of additional repos to add to the dependency solver
ADD_REPO="{{.AdditionalRepos}}"

# # Specify your SUSE registration code for pulling SUSE packages
REG_CODE="{{.RegCode}}"

# # Base directory where all the work will be done
WORK_DIR="{{.WorkDir}}"

# Path to the directory where the zypper repo will be created
REPO_OUT="{{.RepoOut}}"

# Extract the contents of the ISO and extract the raw squashfs
xorriso -osirrox on -indev $WORK_DIR/$ISO_NAME extract / $WORK_DIR/iso-root/
cd $WORK_DIR/iso-root/
unsquashfs $WORK_DIR/iso-root/SLE-Micro.raw.squashfs

# Find the image offset of the rootfs on the raw image
cd squashfs-root
sector=$(/usr/sbin/fdisk -l SLE-Micro.raw | awk '/raw3/ {print $2};')
offset=$(($sector * 512))

# Mount the raw disk image locally to use as a baseline for rpm database
mount -o loop,offset=$offset SLE-Micro.raw $WORK_DIR/root

# Mount the /var btrfs subvolume into the rootfs
loopdev=$(lsblk | grep $WORK_DIR | awk '{print $1}')
mount /dev/$loopdev -o subvol=@/var $WORK_DIR/root/var

# Mount /proc and /sys so that we can ensure that suseconnect commands run
mount -t proc none $WORK_DIR/root/proc
mount -o bind /sys $WORK_DIR/root/sys

# Make the btrfs filesystem read-write on the raw disk image
chroot $WORK_DIR/root btrfs property set / ro false

# Make sure that there would be dns resolution on the raw disk image
cp /etc/resolv.conf $WORK_DIR/root/etc

# TODO: use this once main functionality is there
# Copy the rpms directory locally in the image and make directory for extract
# mkdir -p $WORK_DIR/root/eib-rpms
# mkdir -p $WORK_DIR/root/eib-repo
# cp -r $WORK_DIR/rpms/. $WORK_DIR/root/eib-rpms/

# Append local RPM's to package list and modify directory location
# PKG_LIST="$PKG_LIST $(find $WORK_DIR/rpms -type f)"
# PKG_LIST=$(sed 's|$WORK_DIR/rpms/|/eib-rpms/|g' <<< "$PKG_LIST")

# Make sure that the package list is not empty before proceeding
if [ ${#PKG_LIST} == 1 ];
then
	echo "[ERROR] Package list is empty, and didn't discover any user RPM's"
	exit 22
fi

# Get SLE15 Service Pack number so we enable PackageHub properly
SLE_SP=$(cat $WORK_DIR/root/etc/rpm/macros.sle | awk '/sle/ {print $2};' | cut -c4)

chroot $WORK_DIR/root /bin/bash <<EOF
suseconnect -r $REG_CODE
suseconnect -p PackageHub/15.$SLE_SP/x86_64
zypper ref
counter=1
for i in $ADD_REPO; do
    # Add the additional repositories specified by the user
    zypper ar --no-gpgcheck -f \$i addrepo\$counter 
    counter=\$((counter+1))
done
zypper --pkg-cache-dir /eib-repo/ --no-gpg-checks install -y \
    --download-only --force-resolution --auto-agree-with-licenses \
    --allow-vendor-change -n $PKG_LIST

# Check if the zypper command executed successfully
if [ "\$?" == 0 ]; then
    # This enables already installed packages to force
    # repo generation later, or if there are no additional
    # dependencies required for user-specified rpms
    touch /eib-repo/zypper-success
fi
suseconnect -d
EOF


#TODO: Attempt to move everything below into the Go code

# Copy the RPM's out from the repo created in the container
# cp -rf $WORK_DIR/root/eib-repo/. $WORK_DIR/repo
cp -rf $WORK_DIR/root/eib-repo/. $REPO_OUT

# Clean up the resources created during the dependency process
umount $WORK_DIR/root/proc
umount $WORK_DIR/root/sys
umount $WORK_DIR/root/var
umount $WORK_DIR/root
rm -rf $WORK_DIR/iso-root/ $WORK_DIR/root/ $WORK_DIR/$ISO_NAME

# Check if there has been a failure in the dependency retrieval
if [ ! -z "$PKG_LIST" ] && [ -z "$(ls -A $REPO_OUT)" ];
then
	# PKG_LIST is not empty and the repo directory is empty
	echo "[ERROR] Dependency retrieval unsuccessful, exiting..."
	exit 126
else
	# PKG_LIST is not empty and we have packages in the repo
	echo "[INFO] Dependency retrieval successful."

	# # Copy in the locally specified RPM's
	# mkdir -p $WORK_DIR/repo/local
	# cp $WORK_DIR/rpms/* $WORK_DIR/repo/local/.
	
	# Run createrepo so we can add this to the Combustion phase
	/usr/bin/createrepo $REPO_OUT
	echo "[SUCCESS] Repository successfully created at $REPO_OUT."

	# Strip the path and extension from the list of rpms
	PKG_LIST=$(sed 's|/eib-rpms/||g' <<< "$PKG_LIST")
	PKG_LIST=$(sed 's|.rpm||g' <<< "$PKG_LIST")
	echo $PKG_LIST > $REPO_OUT/package-list.txt
	exit 0
fi