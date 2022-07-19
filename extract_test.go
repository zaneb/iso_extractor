package main

import (
	"testing"

	"os"
)

func TestGetMachineOSImages(t *testing.T) {
	pullspec, err := GetMachineOSImagesPullspec(
		"dockerconfig.json",
		"registry.ci.openshift.org/ocp/release:4.11.0-0.nightly-2022-07-18-173100",
	)
	if err != nil {
		t.Error(err.Error())
	}
	if pullspec != "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:6f0deb29b974061db04b35bb0e6171d7e9a241cc78c6c09bd1a0c10cc3c2f085" {
		t.Errorf("Incorrect pullspec %s", pullspec)
	}
}

func TestExtractISOHash(t *testing.T) {
	outputFile := "coreos-x86_64.iso.sha256"
	os.Remove(outputFile)

	_, err := ExtractISOHash(
		".",
		"dockerconfig.json",
		"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:6f0deb29b974061db04b35bb0e6171d7e9a241cc78c6c09bd1a0c10cc3c2f085",
		"x86_64",
		"",
	)
	if err != nil {
		t.Error(err.Error())
	}
	hash, err := os.ReadFile(outputFile)
	if err != nil {
		t.Error(err.Error())
	}
	if string(hash) != "dff2ceb2e5394f3add6aca6927d244eecfab7f6bcc6079be96c9d3ae79741a3e" {
		t.Errorf("Incorrect hash %s", hash)
	}
}
