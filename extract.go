package main

import (
	"archive/tar"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/openshift/oc/pkg/cli/admin/release"
	"github.com/openshift/oc/pkg/cli/image/archive"
	"github.com/openshift/oc/pkg/cli/image/imagesource"
	imagemanifest "github.com/openshift/oc/pkg/cli/image/manifest"
	"github.com/openshift/oc/pkg/cli/image/strategy"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ISOExtractor struct {
	registryData imagemanifest.SecurityOptions
}

func NewISOExtractor(registryConfigPath string) *ISOExtractor {
	return &ISOExtractor{
		registryData: imagemanifest.SecurityOptions{
			RegistryConfig: registryConfigPath,
		},
	}
}

func (ex *ISOExtractor) GetMachineOSImagesPullspec(releasePullSpec string) (string, error) {
	// oc adm release info --image-for=machine-os-images <release-pullspec>
	// TODO(zaneb): filter by local cpu arch
	inOpts := release.NewInfoOptions(genericclioptions.IOStreams{})
	inOpts.Images = []string{releasePullSpec}
	inOpts.SecurityOptions = ex.registryData

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

func (ex *ISOExtractor) ExtractISOHash(destDir, machineOSImagesPullspec, arch, icspFilePath string) (string, error) {
	// oc image extract --file=/coreos/coreos-x86_64.iso <machine-os-images-pullspec>
	// TODO(zaneb): filter by target cpu arch
	isoPath := strings.TrimLeft(fmt.Sprintf("/coreos/coreos-%s.iso.sha256", arch), "/")

	image := strings.TrimSpace(string(machineOSImagesPullspec))
	imageRef, err := imagesource.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("Invalid pullspec %s: %w", image, err)
	}

	fromContext, err := ex.registryData.Context()
	if err != nil {
		return "", err
	}
	if len(icspFilePath) > 0 {
		fromContext = fromContext.WithAlternateBlobSourceStrategy(strategy.NewICSPOnErrorStrategy(icspFilePath))
	}

	ctx := context.Background()
	fromOpts := &imagesource.Options{
		FileDir:         destDir,
		Insecure:        false,
		RegistryContext: fromContext,
	}
	repo, err := fromOpts.Repository(ctx, imageRef)
	if err != nil {
		return "", fmt.Errorf("unable to connect to image repository %s: %w", imageRef.String(), err)
	}

	filterOptions := imagemanifest.FilterOptions{}
	srcManifest, location, err := imagemanifest.FirstManifest(ctx, imageRef.Ref, repo, filterOptions.Include)
	if err != nil {
		if imagemanifest.IsImageForbidden(err) {
			return "", fmt.Errorf("image %q does not exist or you don't have permission to access the repository: %w", imageRef.String(), err)
		}
		if imagemanifest.IsImageNotFound(err) {
			return "", fmt.Errorf("image %q not found: %w", imageRef.String(), err)
		}
		return "", fmt.Errorf("unable to read image %q: %v", imageRef.String(), err)
	}

	fromBlobs := repo.Blobs(ctx)
	_, layers, err := imagemanifest.ManifestToImageConfig(ctx, srcManifest, fromBlobs, location)
	if err != nil {
		return "", fmt.Errorf("unable to parse image %s: %w", imageRef.String(), err)
	}

	for _, layer := range layers {
		cont, err := func() (bool, error) {
			r, err := fromBlobs.Open(ctx, layer.Digest)
			if err != nil {
				return false, fmt.Errorf("Unable to access the source layer %s: %w", layer.Digest, err)
			}
			defer r.Close()

			options := &archive.TarOptions{
				AlterHeaders: &tarHeaderAlterer{
					dir:  path.Dir(isoPath) + "/",
					name: path.Base(isoPath),
				},
			}

			if _, err := archive.ApplyLayer(".", r, options); err != nil {
				return false, fmt.Errorf("unable to extract layer %s from %s: %w", layer.Digest, imageRef.String(), err)
			}
			return true, nil
		}()
		if err != nil {
			return "", err
		}
		if !cont {
			break
		}
	}

	return filepath.Join(destDir, path.Base(isoPath)), nil
}

type tarHeaderAlterer struct {
	dir  string
	name string
}

func (a *tarHeaderAlterer) Alter(hdr *tar.Header) (bool, error) {
	if !strings.HasPrefix(hdr.Name, a.dir) {
		return false, nil
	}
	hdr.Name = strings.TrimPrefix(hdr.Name, a.dir)
	matchName := hdr.Name
	if i := strings.Index(matchName, "/"); i >= 0 {
		matchName = matchName[:i]
	}
	if ok, err := path.Match(a.name, matchName); !ok || err != nil {
		return false, err
	}
	return true, nil
}
