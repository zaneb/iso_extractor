package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/openshift/oc/pkg/cli/admin/release"
	"github.com/openshift/oc/pkg/cli/image/extract"
	"github.com/openshift/oc/pkg/cli/image/imagesource"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func GetMachineOSImagesPullspec(registryConfigPath, releasePullSpec string) (string, error) {
	output := &bytes.Buffer{}
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    output,
		ErrOut: os.Stderr,
	}

	// oc adm release info --image-for=machine-os-images <release-pullspec>
	// TODO(zaneb): filter by local cpu arch
	inOpts := release.NewInfoOptions(ioStreams)
	inOpts.Images = []string{releasePullSpec}
	inOpts.ImageFor = "machine-os-images"
	inOpts.SecurityOptions.RegistryConfig = registryConfigPath
	inOpts.ShowPullSpec = true

	if err := inOpts.Run(); err != nil {
		return "", fmt.Errorf("failed to get %s image: %w", inOpts.ImageFor, err)
	}
	machineOSImagesPullspec, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(machineOSImagesPullspec)), nil
}

func ExtractISOHash(destDir, registryConfigPath, machineOSImagesPullspec, arch, icspFilePath string) (string, error) {
	// oc image extract --file=/coreos/coreos-x86_64.iso <machine-os-images-pullspec>
	// TODO(zaneb): filter by target cpu arch
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	exOpts := extract.NewExtractOptions(ioStreams)
	isoPath := strings.TrimLeft(fmt.Sprintf("/coreos/coreos-%s.iso.sha256", arch), "/")
	exOpts.Files = []string{isoPath}
	exOpts.ICSPFile = icspFilePath
	exOpts.FileDir = destDir
	exOpts.SecurityOptions.RegistryConfig = registryConfigPath

	image := strings.TrimSpace(string(machineOSImagesPullspec))
	imageRef, err := imagesource.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("Invalid pullspec %s: %w", image, err)
	}

	exOpts.Mappings = []extract.Mapping{
		{
			Image:    image,
			ImageRef: imageRef,
			From:     isoPath,
			To:       ".",
		},
	}

	if err := exOpts.Run(); err != nil {
		return "", fmt.Errorf("failed to extract ISO: %w", err)
	}

	return path.Join(destDir, filepath.Base(isoPath)), nil
}
