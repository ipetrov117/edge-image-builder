package image

import (
	"io"
)

type networkConfigGenerator interface {
	GenerateNetworkConfig(configDir, outputDir string, outputWriter io.Writer) error
}

type networkConfiguratorInstaller interface {
	InstallConfigurator(arch Arch, sourcePath, installPath string) error
}

type kubernetesScriptInstaller interface {
	InstallScript(distribution, sourcePath, destinationPath string) error
}

type kubernetesArtefactDownloader interface {
	DownloadArtefacts(arch Arch, version, cni string, multusEnabled bool, destinationPath string) (installPath, imagesPath string, err error)
}

type LocalRPMConfig struct {
	// LocalPackagesPath is the path to the directory holding RPMs that will be side-loaded
	LocalPackagesPath string
	// GPGKeysPath specifies the path to the directory that holds the GPG keys that the RPMs have been signed with
	GPGKeysPath string
}

type rpmResolver interface {
	Resolve(packages *Packages, localRPMConfig *LocalRPMConfig, outputDir string) (rpmDirPath string, pkgList []string, err error)
}

type rpmRepoCreator interface {
	Create(path string) error
}

type Context struct {
	// ImageConfigDir is the root directory storing all configuration files.
	ImageConfigDir string
	// BuildDir is the directory used for assembling the different components used in a build.
	BuildDir string
	// CombustionDir is a subdirectory under BuildDir containing the Combustion script and all related files.
	CombustionDir string
	// ImageDefinition contains the image definition properties.
	ImageDefinition              *Definition
	NetworkConfigGenerator       networkConfigGenerator
	NetworkConfiguratorInstaller networkConfiguratorInstaller
	KubernetesScriptInstaller    kubernetesScriptInstaller
	KubernetesArtefactDownloader kubernetesArtefactDownloader
	// RPMResolver responsible for resolving rpm/package dependencies
	RPMResolver    rpmResolver
	RPMRepoCreator rpmRepoCreator
}
