// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package rpmrepocloner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microsoft/azurelinux/toolkit/tools/internal/buildpipeline"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/packagerepo/repocloner"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/packagerepo/repomanager/rpmrepomanager"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/pkgjson"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/retry"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/safechroot"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/shell"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/tdnf"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/timestamp"
	"github.com/sirupsen/logrus"
)

// RepoFlag* flags are used to denote which repos the cloner is allowed to use for its queries.
const (
	RepoFlagDistroDefaults  = uint64(1) << iota // External default Azure Linux repos pre-installed in the chroot.
	RepoFlagDownloadedCache                     // Local repo with the cached packages downloaded from upstream.
	RepoFlagLocalBuilds                         // Local repo with the packages built from local spec files.
	RepoFlagPreview                             // Separate flag to control the use of the Azure Linux preview packages repository.
	RepoFlagToolchain                           // Local repo with the toolchain packages.
	RepoFlagUpstream                            // Separate flag to control the use of all upstream packages repositories.

	// A compound flag enabling all supported repositories.
	RepoFlagAll = RepoFlagToolchain | RepoFlagLocalBuilds | RepoFlagDownloadedCache | RepoFlagPreview | RepoFlagDistroDefaults | RepoFlagUpstream
)

const (
	chrootCloneDirContainer = "/upstream-cached-rpms"
	chrootCloneDirRegular   = "/outputrpms"

	repoIDAll            = "*"
	repoIDBuilt          = "local-repo"
	repoIDCacheContainer = "upstream-cache-repo"
	repoIDCacheRegular   = "fetcher-cloned-repo"
	repoIDPreview        = "mariner-preview"
	repoIDToolchain      = "toolchain-repo"

	useSingleTransaction    = true
	useMultipleTransactions = !useSingleTransaction
)

var (
	serverErrorsRegex    = regexp.MustCompile(`(?m)Error: (5\d{2}) when downloading`)
	serverErrorCodeIndex = 1
)

// RpmRepoCloner represents an RPM repository cloner.
type RpmRepoCloner struct {
	chroot                   *safechroot.Chroot
	chrootCloneDir           string
	defaultAzureLinuxRepoIDs []string
	mountedCloneDir          string
	repoSnapshotTime         string
	repoSnapshotArgs         []string
	repoIDCache              string
	reposArgsList            [][]string
	reposFlags               uint64
}

// ConstructCloner constructs a new RpmRepoCloner.
//   - destinationDir is the directory to save RPMs
//   - tmpDir is the directory to create a chroot
//   - workerTar is the path to the worker tar used to seed the chroot
//   - existingRpmsDir is the directory with prebuilt RPMs
//   - prebuiltRpmsDir is the directory with toolchain RPMs
//   - tlsCert is the path to the TLS certificate, "" if not needed
//   - tlsKey is the path to the TLS key, "" if not needed
//   - repoDefinitions is a list of repo files to use
func ConstructCloner(destinationDir, tmpDir, workerTar, existingRpmsDir, toolchainRpmsDir, tlsCert, tlsKey string, repoDefinitions []string, posixTime string) (r *RpmRepoCloner, err error) {
	timestamp.StartEvent("initialize and configure cloner", nil)
	defer timestamp.StopEvent(nil) // initialize and configure cloner

	r = &RpmRepoCloner{}
	err = r.initialize(destinationDir, tmpDir, workerTar, existingRpmsDir, toolchainRpmsDir, repoDefinitions, posixTime)
	if err != nil {
		err = fmt.Errorf("failed to prep new rpm cloner:\n%w", err)
		return
	}

	tlsKey, tlsCert = strings.TrimSpace(tlsKey), strings.TrimSpace(tlsCert)
	err = r.addNetworkFiles(tlsCert, tlsKey)
	if err != nil {
		err = fmt.Errorf("failed to customize RPM repo cloner:\n%w", err)
		return
	}

	return
}

// initialize initializes rpmrepocloner, enabling Clone() to be called.
//   - destinationDir is the directory to save RPMs
//   - tmpDir is the directory to create a chroot
//   - workerTar is the path to the worker tar used to seed the chroot
//   - existingRpmsDir is the directory with prebuilt RPMs
//   - prebuiltRpmsDir is the directory with toolchain RPMs
//   - repoDefinitions is a list of repo files to use when cloning RPMs
func (r *RpmRepoCloner) initialize(destinationDir, tmpDir, workerTar, existingRpmsDir, toolchainRpmsDir string, repoDefinitions []string, posixTime string) (err error) {
	const (
		isExistingDir          = false
		leaveChrootFilesOnDisk = false

		bindFsType = ""
		bindData   = ""

		chrootLocalRpmsDir      = "/localrpms"
		chrootLocalToolchainDir = "/toolchainrpms"

		overlaySource                  = "overlay"
		overlayUpperDirectoryRpms      = "/overlaywork/upper_rpms"
		overlayWorkDirectoryRpms       = "/overlaywork/workdir_rpms"
		overlayUpperDirectoryToolchain = "/overlaywork/upper_toolchain"
		overlayWorkDirectoryToolchain  = "/overlaywork/workdir_toolchain"

		repoFlagClonerDefault = RepoFlagAll & ^RepoFlagPreview
	)

	// Ensure that if initialization fails, the chroot is closed
	defer func() {
		if err != nil {
			logger.Log.Warnf("Failed to initialize cloner:\n%v", err)
			if r.chroot != nil {
				closeErr := r.chroot.Close(leaveChrootFilesOnDisk)
				if closeErr != nil {
					logger.Log.Panicf("Unable to close chroot on failed initialization. Error: %s", closeErr)
				}
			}
		}
	}()

	// Create the directory to download into
	err = os.MkdirAll(destinationDir, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("failed to create download directory (%s):\n%w", destinationDir, err)
		return
	}

	// Setup the chroot
	logger.Log.Infof("Creating cloning environment to populate (%s)", destinationDir)
	r.chroot = safechroot.NewChroot(tmpDir, isExistingDir)

	r.mountedCloneDir = destinationDir

	// Setup mount points for the chroot.
	//
	// 1) Mount the provided directory of existing RPMs into the chroot as an overlay,
	// ensuring the chroot can read the files, but not alter the actual directory outside
	// the chroot.
	//
	// 2) Mount the directory to download RPMs into as a bind, allowing the chroot to write
	// files into it.
	outRpmsOverlayMount, overlayExtraDirs := safechroot.NewOverlayMountPoint(r.chroot.RootDir(), overlaySource, chrootLocalRpmsDir, existingRpmsDir, overlayUpperDirectoryRpms, overlayWorkDirectoryRpms)
	extraMountPoints := []*safechroot.MountPoint{
		outRpmsOverlayMount,
		safechroot.NewMountPoint(destinationDir, chrootCloneDirRegular, bindFsType, safechroot.BindMountPointFlags, bindData),
	}

	// Include the special toolchain packages directory.
	toolchainRpmsOverlayMount, toolchainRpmsOverlayExtraDirs := safechroot.NewOverlayMountPoint(r.chroot.RootDir(), overlaySource, chrootLocalToolchainDir, toolchainRpmsDir, overlayUpperDirectoryToolchain, overlayWorkDirectoryToolchain)
	extraMountPoints = append(extraMountPoints, toolchainRpmsOverlayMount)
	overlayExtraDirs = append(overlayExtraDirs, toolchainRpmsOverlayExtraDirs...)

	// Also request that /overlaywork is created before any chroot mounts happen so the overlay can
	// be created successfully
	err = r.chroot.Initialize(workerTar, overlayExtraDirs, extraMountPoints, true)
	if err != nil {
		r.chroot = nil
		return
	}

	// The 'cacheRepoDir' repo is only used during Docker based builds, which don't
	// use overlay so cache repo must be explicitly initialized.
	// We make sure it's present during all builds to avoid noisy TDNF error messages in the logs.
	reposToInitialize := []string{chrootLocalRpmsDir, chrootCloneDirRegular, chrootCloneDirContainer, chrootLocalToolchainDir}
	for _, repoToInitialize := range reposToInitialize {
		logger.Log.Debugf("Initializing the (%s) repository", repoToInitialize)
		err = r.initializeMountedChrootRepo(repoToInitialize)
		if err != nil {
			return fmt.Errorf("failed to initialize repository (%s):\n%w", repoToInitialize, err)
		}
	}

	logger.Log.Info("Initializing repository configurations")
	err = r.initializeRepoDefinitions(repoDefinitions)
	if err != nil {
		return
	}

	// Docker-based build doesn't use overlay so repo folder
	// must be explicitly set to the RPMs cache folder.
	r.chrootCloneDir = chrootCloneDirContainer
	r.repoIDCache = repoIDCacheContainer
	if buildpipeline.IsRegularBuild() {
		r.chrootCloneDir = chrootCloneDirRegular
		r.repoIDCache = repoIDCacheRegular
	}

	r.SetEnabledRepos(repoFlagClonerDefault)

	if posixTime != "" {
		r.SetRepoEpochTimeLimitArgs(posixTime)
	}

	return
}

// addNetworkFiles adds files needed for networking capabilities into the cloner.
// tlsClientCert and tlsClientKey are optional.
func (r *RpmRepoCloner) addNetworkFiles(tlsClientCert, tlsClientKey string) (err error) {
	files := []safechroot.FileToCopy{
		{Src: "/etc/resolv.conf", Dest: "/etc/resolv.conf"},
	}

	if tlsClientCert != "" && tlsClientKey != "" {
		tlsFiles := []safechroot.FileToCopy{
			{Src: tlsClientCert, Dest: "/etc/tdnf/mariner_user.crt"},
			{Src: tlsClientKey, Dest: "/etc/tdnf/mariner_user.key"},
		}

		files = append(files, tlsFiles...)
	}

	err = r.chroot.AddFiles(files...)
	return
}

// initializeRepoDefinitions will configure the chroot's repo files to match those
// provided by the caller.
func (r *RpmRepoCloner) initializeRepoDefinitions(repoDefinitions []string) (err error) {
	// ============== TDNF SPECIFIC IMPLEMENTATION ==============
	// Unlike some other package managers, TDNF has no notion of repository priority.
	// It reads the repo files using `readdir`, which should be assumed to be random ordering.
	//
	// In order to simulate repository priority, concatenate all requested repofiles into a single file.
	// TDNF will read the file top-down. It will then parse the results into a linked list, meaning
	// the first repo entry in the file is the first to be checked.
	const (
		chrootRepoDir  = "/etc/yum.repos.d/"
		chrootRepoFile = "allrepos.repo"
	)

	fullRepoDirPath := filepath.Join(r.chroot.RootDir(), chrootRepoDir)
	fullRepoFilePath := filepath.Join(fullRepoDirPath, chrootRepoFile)

	// Create the directory for the repo file in case there wasn't already one there
	err = os.MkdirAll(filepath.Dir(fullRepoFilePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory for chroot repo file (%s):\n%w", fullRepoFilePath, err)
	}

	// Get a list of the existing repofiles that are part of the chroot, if any
	// We need to capture this list before we add 'allrepos.repo'.
	existingRepoFiles, err := filepath.Glob(filepath.Join(fullRepoDirPath, "*"))
	if err != nil {
		return fmt.Errorf("failed to list existing repo files (%s):\n%w", fullRepoDirPath, err)
	}

	dstFile, err := os.OpenFile(fullRepoFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return
	}
	defer dstFile.Close()

	// Append all repo files together into a single repo file.
	// Assume the order of repoDefinitions indicates their relative priority.
	for _, repoFilePath := range repoDefinitions {
		err = appendRepoFile(repoFilePath, dstFile)
		if err != nil {
			return
		}
	}

	// Add each previously existing repofile to the end of the new file, then delete the original.
	// We want to try our custom mounted repos before reaching out to the upstream servers.
	// By default, chroot ships with PMC repositories specified in azurelinux-repos rpm.
	for _, originalRepoFilePath := range existingRepoFiles {
		repoIDs, err := readRepoIDs(originalRepoFilePath)
		if err != nil {
			return err
		}
		r.defaultAzureLinuxRepoIDs = append(r.defaultAzureLinuxRepoIDs, repoIDs...)

		err = appendRepoFile(originalRepoFilePath, dstFile)
		if err != nil {
			return err
		}
		err = os.Remove(originalRepoFilePath)
		if err != nil {
			return err
		}
	}

	return
}

func appendRepoFile(repoFilePath string, dstFile *os.File) (err error) {
	repoFile, err := os.Open(repoFilePath)
	if err != nil {
		return
	}
	defer repoFile.Close()

	_, err = io.Copy(dstFile, repoFile)
	if err != nil {
		return
	}

	// Append a new line
	_, err = dstFile.WriteString("\n")
	return
}

// initializeMountedChrootRepo will initialize a local RPM repository inside the chroot.
func (r *RpmRepoCloner) initializeMountedChrootRepo(repoDir string) (err error) {
	return r.chroot.Run(func() (err error) {
		err = os.MkdirAll(repoDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create repo directory (%s):\n%w", repoDir, err)
		}
		err = rpmrepomanager.CreateRepo(repoDir)
		if err != nil {
			return fmt.Errorf("failed to create an RPM repository under (%s):\n%w", repoDir, err)
		}

		return r.refreshPackagesCache()
	})
}

// Clone clones the provided list of packages.
// If cloneDeps is set, package dependencies will also be cloned.
// It will automatically resolve packages that describe a provide or file from a package.
// If all packages were pre-built, the cloner will set allPackagesPrebuilt = true.
func (r *RpmRepoCloner) CloneByPackageVer(cloneDeps bool, packagesToClone ...*pkgjson.PackageVer) (allPackagesPrebuilt bool, err error) {
	packageNames := []string{}
	for _, packageToClone := range packagesToClone {
		logger.Log.Debugf("Cloning (%s).", packageToClone)
		packageNames = append(packageNames, convertPackageVersionToTdnfArg(packageToClone))
	}
	return r.CloneByName(cloneDeps, packageNames...)
}

// CloneTransaction clones the provided list of packages in a single transaction.
// If cloneDeps is set, package dependencies will also be cloned.
// It will automatically resolve packages that describe a provide or file from a package.
// If all packages were pre-built, the cloner will set allPackagesPrebuilt = true.
func (r *RpmRepoCloner) CloneByPackageVerSingleTransaction(cloneDeps bool, packagesToClone ...*pkgjson.PackageVer) (allPackagesPrebuilt bool, err error) {
	packageNames := []string{}
	for _, packageToClone := range packagesToClone {
		logger.Log.Debugf("Cloning (%s).", packageToClone)
		packageNames = append(packageNames, convertPackageVersionToTdnfArg(packageToClone))
	}
	return r.CloneByNameSingleTransaction(cloneDeps, packageNames...)
}

// CloneByName clones the provided package name exactly as specified, with one transaction per package.
// Any conditional strings will be passed to tdnf verbatim (i.e. "pkg = ver1.2.3-4.cm2")
// If cloneDeps is set, package dependencies will also be cloned.
// This version of clone will not resolve provides or files from other packages beyond what tdnf is able to do itself.
// If all packages were pre-built, the cloner will set allPackagesPrebuilt = true.
func (r *RpmRepoCloner) CloneByName(cloneDeps bool, rawPackageNames ...string) (allPackagesPrebuilt bool, err error) {
	return r.cloneRawPackageNames(cloneDeps, useMultipleTransactions, rawPackageNames...)
}

// CloneRawPackageNames clones the provided package name exactly as specified, using a single transaction.
// Any conditional strings will be passed to tdnf verbatim (i.e. "pkg = ver1.2.3-4.cm2")
// If cloneDeps is set, package dependencies will also be cloned.
// This version of clone will not resolve provides or files from other packages beyond what tdnf is able to do itself.
// If all packages were pre-built, the cloner will set allPackagesPrebuilt = true.
func (r *RpmRepoCloner) CloneByNameSingleTransaction(cloneDeps bool, rawPackageNames ...string) (allPackagesPrebuilt bool, err error) {
	return r.cloneRawPackageNames(cloneDeps, useSingleTransaction, rawPackageNames...)
}

// cloneRawPackageNames clones the requested packages by name exactly as requested (including any version or condition).
// If cloneDeps is set, package dependencies will also be cloned.
// If singleTransaction is set, all packages will be cloned in a single transaction.
// If all packages come from the toolchain or local builds, the cloner will set allPackagesPrebuilt = true.
func (r *RpmRepoCloner) cloneRawPackageNames(cloneDeps, singleTransaction bool, rawPackageNames ...string) (allPackagesPrebuilt bool, err error) {
	timestamp.StartEvent("cloning packages", nil)
	defer timestamp.StopEvent(nil)

	depsSwitch := "--nodeps"
	if cloneDeps {
		depsSwitch = "--alldeps"
	}

	constantArgs := []string{
		"install",
		"-y",
		depsSwitch,
		"--downloadonly",
		"--downloaddir",
		r.chrootCloneDir,
	}

	if r.GetRepoSnapshotTime() != "" {
		constantArgs = append(constantArgs, r.GetRepoSnapshotArgs()...)
	}

	logger.Log.Debugf("Will clone in total %d items.", len(rawPackageNames))

	// Create a list of lists for each transaction. Each transaction will be cloned separately. Generally either all
	// packages will be cloned in a single transaction or each package will be cloned in its own transaction.
	transactions := [][]string{}
	if singleTransaction {
		transactions = append(transactions, rawPackageNames)
	} else {
		for _, packageName := range rawPackageNames {
			transactions = append(transactions, []string{packageName})
		}
	}

	allPackagesPrebuilt = true
	for _, packageNamesToClone := range transactions {
		logger.Log.Debugf("Cloning raw names (%v).", packageNamesToClone)

		finalArgs := append(constantArgs, packageNamesToClone...)
		err = r.chroot.Run(func() (chrootErr error) {
			prebuilt, chrootErr := r.clonePackage(finalArgs)
			if !prebuilt {
				allPackagesPrebuilt = false
			}
			return
		})

		if err != nil {
			return
		}
	}

	return
}

// WhatProvides attempts to find packages which provide the requested PackageVer.
func (r *RpmRepoCloner) WhatProvides(pkgVer *pkgjson.PackageVer) (packageNames []string, err error) {
	var (
		releaseverCliArg string
	)

	releaseverCliArg, err = tdnf.GetReleaseverCliArg()
	if err != nil {
		return
	}

	provideQuery := convertPackageVersionToTdnfArg(pkgVer)

	baseArgs := []string{
		"provides",
		provideQuery,
		releaseverCliArg,
	}

	if r.GetRepoSnapshotTime() != "" {
		baseArgs = append(baseArgs, r.GetRepoSnapshotArgs()...)
	}

	// Consider the built (toolchain, local) RPMs first, then the already cached, and finally all remote packages.
	for _, reposArgs := range r.reposArgsList {
		logger.Log.Debugf("Using repos args: %v", reposArgs)

		err = r.chroot.Run(func() (err error) {
			completeArgs := append(baseArgs, reposArgs...)

			stdout, stderr, err := shell.Execute("tdnf", completeArgs...)
			logger.Log.Debugf("tdnf search for provide '%s':\n%s", pkgVer.Name, stdout)

			if err != nil {
				logger.Log.Debugf("Failed to lookup provide '%s', tdnf error: '%s'", pkgVer.Name, stderr)
				return
			}

			// Flag the result if we did case-insensitive lookup.
			lookupIgnoredCase := tdnf.DidCaseInsensitiveMatchRegex.MatchString(stdout)

			// MUST keep order of packages printed by TDNF.
			// TDNF will print the packages starting from the highest version, which allows us to work around an RPM bug:
			// https://github.com/rpm-software-management/rpm/issues/2359
			for _, matches := range tdnf.PackageProvidesRegex.FindAllStringSubmatch(stdout, -1) {
				packageName := matches[tdnf.PackageProvidesNameIndex]
				if lookupIgnoredCase {
					logger.Log.Warnf("'%s' was found by case-insensitive lookup of '%s', but this is not valid and will be ignored", packageName, pkgVer.Name)
					// This is not a valid mapping of requires -> provides, so we skip it. This is not a fatal error since
					// we may still find a valid mapping either in a subsequent repo lookup, or via a local build. If this
					// is infact invalid, the package build will fail later with an unresolved dependency error.
					return
				}
				packageNames = append(packageNames, packageName)
				logger.Log.Debugf("'%s' is available from package '%s'", pkgVer.Name, packageName)
			}

			return
		})
		if err != nil {
			return
		}

		if len(packageNames) > 0 {
			logger.Log.Debug("Found required package(s), skipping further search in other repos.")
			break
		}
	}

	if len(packageNames) == 0 {
		err = fmt.Errorf("could not resolve %s", pkgVer.Name)
		return
	}

	logger.Log.Debugf("Translated '%s' to package(s): %s", pkgVer.Name, strings.Join(packageNames, " "))
	return
}

// ConvertDownloadedPackagesIntoRepo initializes the downloaded RPMs into an RPM repository.
// Packages will be placed in a flat directory.
func (r *RpmRepoCloner) ConvertDownloadedPackagesIntoRepo() (err error) {
	logger.Log.Info("Configuring downloaded RPMs as a local repository")
	timestamp.StartEvent("covert packages to repo", nil)
	defer timestamp.StopEvent(nil)

	err = r.initializeMountedChrootRepo(chrootCloneDirRegular)
	if err != nil {
		return
	}

	repoDir := filepath.Join(r.chroot.RootDir(), r.chrootCloneDir)

	// Print warnings for any invalid RPMs
	err = rpmrepomanager.ValidateRpmPaths(repoDir)
	if err != nil {
		logger.Log.Warnf("Failed to validate RPM paths: %s", err)
		// We treat this as just a warning, not a real error.
		err = nil
	}

	if !buildpipeline.IsRegularBuild() {
		// Docker based build doesn't use overlay so cache repo
		// must be explicitly initialized
		err = r.initializeMountedChrootRepo(chrootCloneDirContainer)
		if err != nil {
			return
		}
	}

	return
}

// ClonedRepoContents returns the non-local, downloaded packages.
// This includes the toolchain packages along with other packages downloaded from the upstream repositories.
func (r *RpmRepoCloner) ClonedRepoContents() (repoContents *repocloner.RepoContents, err error) {
	releaseverCliArg, err := tdnf.GetReleaseverCliArg()
	if err != nil {
		return
	}

	// We may hit duplicates between the toolchain packages and the other upstream ones
	// and we don't want to list them twice.
	foundPackages := map[string]bool{}
	repoContents = &repocloner.RepoContents{}
	onStdout := func(line string) {
		matches := tdnf.ListedPackageRegex.FindStringSubmatch(line)
		if len(matches) != tdnf.ListedPackageMaxMatchLen {
			return
		}

		pkg := &repocloner.RepoPackage{
			Name:         matches[tdnf.ListedPackageName],
			Version:      matches[tdnf.ListedPackageVersion],
			Architecture: matches[tdnf.ListedPackageArch],
			Distribution: matches[tdnf.ListedPackageDist],
		}

		pkgID := pkg.ID()
		if foundPackages[pkgID] {
			logger.Log.Debugf("Skipping duplicate package: %s", line)
			return
		}
		foundPackages[pkgID] = true

		logger.Log.Debugf("Found package: %s", line)

		repoContents.Repo = append(repoContents.Repo, pkg)
	}

	// We only enable the cache repo, but TDNF will also always list the '@System' packages.
	err = r.chroot.Run(func() (err error) {
		tdnfArgs := []string{
			"list",
			"ALL",
			fmt.Sprintf("--disablerepo=%s", repoIDAll),
			fmt.Sprintf("--enablerepo=%s", r.repoIDCache),
			releaseverCliArg,
		}

		return shell.NewExecBuilder("tdnf", tdnfArgs...).
			StdoutCallback(onStdout).
			LogLevel(logrus.TraceLevel, logrus.WarnLevel).
			WarnLogLines(shell.DefaultWarnLogLines).
			Execute()
	})

	return
}

// CloneDirectory returns the directory where cloned packages are saved.
func (r *RpmRepoCloner) CloneDirectory() string {
	return r.mountedCloneDir
}

// Close closes the given RpmRepoCloner.
func (r *RpmRepoCloner) Close() error {
	const leaveChrootFilesOnDisk = false
	return r.chroot.Close(leaveChrootFilesOnDisk)
}

// clonePackage clones a given package using pre-populated arguments.
// It will gradually enable more repos to consider until the package is found.
func (r *RpmRepoCloner) clonePackage(baseArgs []string) (preBuilt bool, err error) {

	releaseverCliArg, err := tdnf.GetReleaseverCliArg()
	if err != nil {
		return
	}

	baseArgs = append(baseArgs, releaseverCliArg)

	for _, reposArgs := range r.reposArgsList {
		logger.Log.Debugf("Using repo args: %s", reposArgs)

		finalArgs := append(baseArgs, reposArgs...)

		// We run in a retry loop on errors deemed retriable.
		ctx, closeCtx := context.WithCancel(context.Background())
		defer closeCtx()

		retryNum := 1
		_, err = retry.RunWithDefaultDownloadBackoff(ctx, func() error {
			downloadErr, retriable := tdnfDownload(finalArgs...)
			if downloadErr != nil {
				if retriable {
					logger.Log.Debugf("Package cloning attempt %d/%d failed with a retriable error.", retryNum, retry.DefaultDownloadRetryAttempts)
				} else {
					logger.Log.Debugf("Package cloning attempt %d/%d failed with an unrecoverable error. Cancelling.", retryNum, retry.DefaultDownloadRetryAttempts)
					closeCtx()
				}
			}

			retryNum++
			return downloadErr
		})

		if err == nil {
			preBuilt = r.reposArgsHaveOnlyLocalSources(reposArgs)
			break
		}
	}

	return
}

func convertPackageVersionToTdnfArg(pkgVer *pkgjson.PackageVer) (tdnfArg string) {
	tdnfArg = pkgVer.Name

	// TDNF does not accept versioning information on implicit provides.
	if pkgVer.IsImplicitPackage() {
		if pkgVer.Condition != "" {
			logger.Log.Warnf("Discarding version constraint for implicit package: %v", pkgVer)
		}
		return
	}

	// To avoid significant overhead we only download the latest version of a package
	// for ">" and ">=" constraints (ie remove constraints).
	switch pkgVer.Condition {
	case "":
	case "=":
		tdnfArg = fmt.Sprintf("%s-%s", pkgVer.Name, pkgVer.Version)
	case "<=", "<":
		tdnfArg = fmt.Sprintf("%s %s %s", pkgVer.Name, pkgVer.Condition, pkgVer.Version)
	case ">", ">=":
		logger.Log.Warnf("Discarding '%s' version constraint for: %v", pkgVer.Condition, pkgVer)
	default:
		logger.Log.Errorf("Unsupported version constraint: %s", pkgVer.Condition)
	}

	return
}

func (r *RpmRepoCloner) GetRepoSnapshotTime() string {
	return r.repoSnapshotTime
}

func (r *RpmRepoCloner) GetRepoSnapshotArgs() []string {
	return r.repoSnapshotArgs
}

func (r *RpmRepoCloner) SetRepoEpochTimeLimitArgs(posixTime string) {
	var (
		snapshotTimeArg    string
		snapshotExcludeArg string
		excludeRepoIds     []string
		err                error
	)

	r.repoSnapshotTime = posixTime
	r.repoSnapshotArgs = []string{}

	if r.repoSnapshotTime == "" { // no args to add
		return
	}

	snapshotTimeArg, err = tdnf.GetRepoSnapshotCliArg(r.repoSnapshotTime)
	if err != nil {
		logger.Log.Errorf("Snapshot Time is invalid")
		return
	}
	excludeRepoIds = []string{repoIDBuilt, repoIDToolchain, r.repoIDCache, repoIDCacheRegular}
	snapshotExcludeArg, err = tdnf.GetRepoSnapshotExcludeCliArg(excludeRepoIds)
	if err != nil {
		logger.Log.Errorf("Snapshot Repo to exclude is invalid")
		return
	}

	r.repoSnapshotArgs = append(r.repoSnapshotArgs, snapshotTimeArg, snapshotExcludeArg)
}

// GetEnabledRepos returns the repo flags that the cloner is allowed to use for its queries.
func (r *RpmRepoCloner) GetEnabledRepos() uint64 {
	return r.reposFlags
}

// SetEnabledRepos tells the cloner which repos it is allowed to use for its queries.
func (r *RpmRepoCloner) SetEnabledRepos(reposFlags uint64) {
	r.reposFlags = reposFlags
	r.reposArgsList = [][]string{}
	previousReposList := []string{fmt.Sprintf("--disablerepo=%s", repoIDAll)}

	defer func() {
		logger.Log.Debugf("Enabled repos: %v.", r.reposArgsList)
	}()

	// Do NOT change the order of the following 'if' statements!
	// The order is critical as we want to gradually enable repositories in the following order:
	// 1. Toolchain.
	// 2. Locally-built packages.
	// 3. Local cache of packages downloaded from external sources.
	// 4. Upstream repositories requiring network access.
	if RepoFlagToolchain&reposFlags != 0 {
		previousReposList = append(previousReposList, fmt.Sprintf("--enablerepo=%s", repoIDToolchain))
		r.reposArgsList = append(r.reposArgsList, previousReposList)
	}

	if RepoFlagLocalBuilds&reposFlags != 0 {
		previousReposList = append(previousReposList, fmt.Sprintf("--enablerepo=%s", repoIDBuilt))
		r.reposArgsList = append(r.reposArgsList, previousReposList)
	}

	if RepoFlagDownloadedCache&reposFlags != 0 {
		previousReposList = append(previousReposList, fmt.Sprintf("--enablerepo=%s", r.repoIDCache))
		r.reposArgsList = append(r.reposArgsList, previousReposList)
	}

	// Options past this point are only valid if upstream repos are enabled.
	if RepoFlagUpstream&reposFlags == 0 {
		return
	}

	previousReposList = append(previousReposList, fmt.Sprintf("--enablerepo=%s", repoIDAll))

	if RepoFlagPreview&reposFlags == 0 {
		previousReposList = append(previousReposList, fmt.Sprintf("--disablerepo=%s", repoIDPreview))
	}

	if RepoFlagDistroDefaults&reposFlags == 0 {
		previousReposList = append(previousReposList, r.disabledDefaultAzureLinuxReposArgs()...)
	}

	r.reposArgsList = append(r.reposArgsList, previousReposList)
}

func (r *RpmRepoCloner) disabledDefaultAzureLinuxReposArgs() (args []string) {
	args = make([]string, len(r.defaultAzureLinuxRepoIDs))
	for i, repoID := range r.defaultAzureLinuxRepoIDs {
		args[i] = fmt.Sprintf("--disablerepo=%s", repoID)
	}

	return
}

func (r *RpmRepoCloner) refreshPackagesCache() (err error) {
	releaseverCliArg, err := tdnf.GetReleaseverCliArg()
	if err != nil {
		return
	}

	args := []string{
		"makecache",
		releaseverCliArg,
		fmt.Sprintf("--enablerepo=%s", repoIDAll),
	}

	stdout, stderr, err := shell.Execute("tdnf", args...)
	if err != nil {
		logger.Log.Errorf("Failed to run 'tdnf makecache'\nstdout:\n%v", stdout)
		return fmt.Errorf("failed to run 'tdnf makecache':\n%v\n%w", stderr, err)
	}

	return
}

func readRepoIDs(repoFilePath string) (repoIDs []string, err error) {
	repoFile, err := os.Open(repoFilePath)
	if err != nil {
		return
	}
	defer repoFile.Close()

	scanner := bufio.NewScanner(repoFile)
	for scanner.Scan() {
		matches := tdnf.RepoIDRegex.FindStringSubmatch(scanner.Text())
		if len(matches) <= tdnf.RepoIDIndex {
			continue
		}

		repoID := matches[tdnf.RepoIDIndex]
		repoIDs = append(repoIDs, repoID)

		logger.Log.Debugf("Found repo ID: %s", repoID)
	}

	err = scanner.Err()
	return

}

func (r *RpmRepoCloner) reposArgsHaveOnlyLocalSources(reposArgs []string) bool {
	const repoIDIndex = 1

	for _, repoArg := range reposArgs {
		if strings.Contains(repoArg, "--enablerepo=") {
			repoID := strings.Split(repoArg, "=")[repoIDIndex]
			if repoID != repoIDBuilt && repoID != repoIDToolchain {
				return false
			}
		}
	}

	return true
}

func tdnfDownload(args ...string) (err error, retriable bool) {
	const (
		unresolvedOutputPrefix = "No package"
		unresolvedOutputSuffix = "available"
	)

	stdout, stderr, err := shell.Execute("tdnf", args...)

	logger.Log.Debugf("stdout: %s", stdout)
	logger.Log.Debugf("stderr: %s", stderr)

	// ============== TDNF SPECIFIC IMPLEMENTATION ==============
	//
	// Check if TDNF could not resolve a given package. If TDNF does not find a requested package,
	// it will not error. Instead it will print a message to stdout. Check for this message.
	//
	// *NOTE*: TDNF will attempt best effort. If N packages are requested, and 1 cannot be found,
	// it will still download N-1 packages while also printing the message.
	splitStdout := strings.Split(stdout, "\n")
	for _, line := range splitStdout {
		trimmedLine := strings.TrimSpace(line)
		// If a package was not available, update err
		if strings.HasPrefix(trimmedLine, unresolvedOutputPrefix) && strings.HasSuffix(trimmedLine, unresolvedOutputSuffix) {
			err = fmt.Errorf(trimmedLine)
			return
		}
	}

	//
	// *NOTE*: There are cases in which some of our upstream package repositories are hosted
	// on services that are prone to intermittent errors (e.g., HTTP 502 errors). We
	// specifically look for such known cases and apply some retry logic in hopes of getting
	// a better result; note that we don't indiscriminately retry because there are legitimate
	// cases in which the upstream repo doesn't contain the package and a 404 error is to be
	// expected. This involves scraping through stderr, but it's better than not doing so.
	//
	if err != nil {
		serverErrorMatch := serverErrorsRegex.FindStringSubmatch(stderr)
		if len(serverErrorMatch) > serverErrorCodeIndex {
			logger.Log.Debugf("Encountered possibly intermittent HTTP %s error.", serverErrorMatch[serverErrorCodeIndex])
			retriable = true
		}
	}

	return
}
