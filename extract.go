package main

import (
	"fmt"
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
	// oc adm release info --image-for=machine-os-images <release-pullspec>
	// TODO(zaneb): filter by local cpu arch
	inOpts := release.NewInfoOptions(genericclioptions.IOStreams{})
	inOpts.Images = []string{releasePullSpec}
	inOpts.SecurityOptions.RegistryConfig = registryConfigPath

	release, err := inOpts.LoadReleaseInfo(releasePullSpec, false)
	if err != nil {
		return "", fmt.Errorf("failed to load release info")
	}

	for _, tag := range release.References.Spec.Tags {
		if tag.Name == "machine-os-images" {
			if tag.From != nil && tag.From.Kind == "DockerImage" && len(tag.From.Name) > 0 {
				return tag.From.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no machine-os-images image exists in release image %s", releasePullSpec)
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
