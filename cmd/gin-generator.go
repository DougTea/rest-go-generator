package cmd

import (
	"github.com/DougTea/rest-go-generator/pkg/gin"
	"golang.org/x/tools/go/packages"
	"k8s.io/gengo/args"
	"k8s.io/gengo/namer"
	"k8s.io/klog/v2"
)

func Run() {
	klog.InitFlags(nil)
	arguments := args.Default()

	// Override defaults.
	arguments.OutputFileBaseName = "Controller"
	arguments.OutputBase = "."
	if cpkg, err := packages.Load(&packages.Config{Mode: packages.NeedName}, "."); err == nil && len(cpkg) == 1 {
		arguments.InputDirs = append(arguments.InputDirs, cpkg[0].PkgPath)
	} else {
		klog.Fatalf("Error: %v", err)
	}
	// Run it.
	if err := arguments.Execute(
		map[string]namer.Namer{
			"public": namer.NewPublicNamer(0),
		},
		"public",
		gin.Packages,
	); err != nil {
		klog.Fatalf("Error: %v", err)
	}
	klog.V(2).Info("Completed successfully.")
}
