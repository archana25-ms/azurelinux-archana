# Azure Linux Image Customizer command line

## --help

Displays the tool's quick help.

## --build-dir=DIRECTORY-PATH

Required.

The directory where the tool will place its temporary files.

## --image-file=FILE-PATH

Required.

The base image file to customize.

This file is typically one of the standard Azure Linux core images.
But it can also be an Azure Linux image that has been customized.

Supported image file formats: vhd, vhdx, qcow2, and raw.

## --output-image-file=FILE-PATH

Required.

The file path to write the final customized image to.

## --output-image-format=FORMAT

The image format of the the final customized image.

Options: vhd, vhd-fixed, vhdx, qcow2, raw, and iso.

At least one of `--output-image-format` and `--output-split-partitions-format` is 
required.

The vhd-fixed option outputs a fixed size VHD image. This is the required format for
VMs in Azure.

When the output image format is set to iso, the generated image is a LiveOS
iso image. For more details on this format, see: 
[Azure Linux Image Customizer ISO Support](./iso.md).

## --output-split-partitions-format=FORMAT

Format of partition files. If specified, disk partitions will be extracted as separate
files and a json file with partition metadata will be produced. For more details on 
the json file format, see: [Partition Metadata JSON Format](./partitionmetadatajson.md).

Options: raw, raw-zst.

## --shrink-filesystems

Enable shrinking of partition filesystems to their minimum size.

Currently only supports ext2/ext3/ext4 filesystems.

Can only be specified if `--output-split-partitions-format` is, and 
cannot be specified with `--output-image-format`.

## --config-file=FILE-PATH

Required.

The file path of the YAML (or JSON) configuration file that specifies how to customize
the image.

For documentation on the supported configuration options, see:
[Azure Linux Image Customizer configuration](./docs/configuration.md)

## --rpm-source=PATH

A resource that provides RPM files to be used during package installation.

Can be one of:

- Directory path: A path to a directory containing RPM files.

  The RPMs may either be in the directory itself or any subdirectories.

- `*.repo` file path: A path to a RPM repo definition file.

  The file name extension must be `.repo`.

  Note: This file is not installed in the image during customization.
  If that is also needed, then use `AdditionalFiles` to place the repo file within
  the image.

This option can be specified multiple times.

RPM sources are specified in the order or priority from lowest to highest.
If `--disable-base-image-rpm-repos` is not specified, then the in-built RPM repos are
given the lowest priority.

See, [Building custom packages](building-packages.md) for a guide on how to build your
own packages for Azure Linux.

See, [Cloning an RPM repo](clone-rpm-repo.md) for how to clone or download RPMs from a
existing RPM repo (such as packages.microsoft.com). Using a cloned repo with
`--rpm-source` can help your builds avoid dependencies on external resources.

## --disable-base-image-rpm-repos

Disable the base image's installed RPM repos as a source of RPMs during package
installation.

## --output-pxe-artifacts-dir

Create a folder containing the artifacts to be used for PXE booting.

For an overview of Azure Linux Image Customizer support for PXE, see the 
[PXE support page](./pxe.md).

## --log-level=LEVEL

Default: `info`

The verbosity of logs the tool outputs.

Higher levels of logging may be useful for debugging what the tool is doing.

The levels from lowest to highest level of verbosity are: `panic`, `fatal`, `error`,
`warn`, `info`, `debug`, and `trace`.
