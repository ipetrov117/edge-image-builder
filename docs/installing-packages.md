# Installing packages
This documentation aims at explaining how a user can configure packages for installation and dependency resolution. For a deep dive on the architecture and workflow of EIB's RPM resolver, see the [RPM resolver architecture](rpm-resolver-architecture.md) documentation.

## Supported systems
The RPM package dependency resolution and installation has been tested on the following systems:  
1. Ubuntu 22.04
2. SLES 15-SP5
3. openSUSE Tumbleweed 
4. Fedora Linux

## Running the EIB container
In order for EIB to resolve the dependencies for the packages that you provided, it starts a podman container within its container. To do this, the EIB container needs to be started with the adequate permissions.  

In this case the `--privileged` needs to be provided to the standart EIB run command:
```shell
podman run --rm --privileged -it \
-v $IMAGE_DIR:/eib eib:dev /bin/eib \
-config-file $CONFIG_FILE.yaml \
-config-dir /eib \
-build-dir /eib/_build
```

> **_NOTE:_** Depending on the `cgroupVersion` that podman operates with, you might also need to run the command with `sudo` permissions. This is the case for `cgroupVersion: v1`, mainly because the `--privileged` option does not support non-root usage for this version. For `cgroupVersion: v2`, you can run the command without root permissions. In order to check the `cgroupVersion` that podman operates with, run this command: `podman info | grep cgroupVersion`.

## Configuring packages for installation
You can configure packages for installation in the following ways:
1. provide a `packageList` configuration under `operatingSystem.packages` in the EIB image configuration file
2. create an `rpms` directory under EIB's configuration directory and provide local RPM files that you wish to be resolved and installed

### Installing packages through 'packageList'
To install a package using the `packageList` at a minimum you must configure the following under `operatingSystem.packages`:
1. valid package names under `packageList`
2. either an `additionalRepo` or a `registrationCode` provided

#### Installing a package from a third-party repo
```yaml
operatingSystem:
  packages:
    packageList:
      - reiserfs-kmp-default-debuginfo
    additionalRepos:
      - url: https://download.opensuse.org/repositories/Kernel:/SLE15-SP5/pool
```
> **_NOTE:_** Before adding any repositories under `additionalRepos`, make sure that they are signed with a valid GPG key. If unsigned repositories are added the EIB package resolution will fail.

#### Installing a package from SUSE's internal repositories
```yaml
operatingSystem:
  packages:
    packageList:
      - wget2
    registrationCode: <your-reg-code>
```

### Side-loading RPMs
Sometimes you may want to install RPM files that are local to your machine.  
You can do this by creating a directory called `rpms` under the EIB configuration directory and copying your local RPM files to this directory.

If your RPM is dependent on other packages, then you must provide either an entry under `additionalRepos` or you must provide the `registrationCode` property.

> **_NOTE:_** All RPMs that will be side-loaded must have valid GPG signatures. The GPG keys used to sign the RPMs must be copied to the `gpg-keys` directory which must be created under `<eib-config-dir>/rpms`. If you try to install RPMs that are unsgined or have unrecognized GPG keys, the EIB package resolution will fail.

#### RPM with dependency resolution from a third-party repository  
EIB configuration directory tree:
```shell
.
├── eib-config-iso.yaml
├── images
│   └── SLE-Micro.x86_64-5.5.0-Default-RT-GM.raw
└── rpms
    ├── gpg-keys
    │   └── reiserfs-kpm-default-debuginfo.key
    └── reiserfs-kmp-default-debuginfo-5.14.21-150500.205.1.g8725a95.x86_64.rpm
```

EIB config file `packages` configuration:
```yaml
operatingSystem:
  packages:
    registrationCode:
      - url: https://download.opensuse.org/repositories/Kernel:/SLE15-SP5/pool
```

#### RPM with depdendency resolution from SUSE's internal repositories
EIB configuration directory tree:
```shell
.
├── eib-config-iso.yaml
├── images
│   └── SLE-Micro.x86_64-5.5.0-Default-RT-GM.raw
└── rpms
    ├── gpg-keys
    │   └── git.key
    └── git-2.35.3-150300.10.33.1.x86_64.rpm
```

EIB config file `packages` configuration:
```yaml
operatingSystem:
  packages:
    additionalRepos: <your-reg-code>
```

### Installing unsigned packages
By default EIB does GPG validation for every additional repository and every side-loaded RPM. If you wish to use either unsigned additional repositories or unsinged RPMs you must provide the `noGPGCheck: true` property in the `packages` configuration, like so:
```yaml
operatingSystem:
  packages:
    noGPGCheck: true
```
By doing this **all** GPG validation will be disabled and you will be able to add unsigned additional repositories or RPMs.

> **_NOTE:_** This property is intended for development use only. For production use-cases we encourage users to not disable EIB's GPG validation.