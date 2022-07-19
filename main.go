package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/klog/v2"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(fs)
	fs.Parse([]string{"-v=5"})

	machineOSImagesPullspec, err := GetMachineOSImagesPullspec(
		"dockerconfig.json",
		"registry.ci.openshift.org/ocp/release:4.11.0-0.nightly-2022-07-18-173100",
	)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	_, err = ExtractISOHash(
		".",
		"dockerconfig.json",
		machineOSImagesPullspec,
		"x86_64", // TODO(zaneb): Don't hard-code arch
		"",       // TODO(zaneb): Pass ICSP tempfile
	)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(2)
	}
}
