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

	_, err := ExtractISOFile(
		".",
		"dockerconfig.json",
		"registry.ci.openshift.org/ocp/release:4.11.0-0.nightly-2022-07-18-173100",
		"x86_64", // TODO(zaneb): Don't hard-code arch
		"",       // TODO(zaneb): Pass ICSP tempfile
	)

	fmt.Printf("%v\n", err)
}
