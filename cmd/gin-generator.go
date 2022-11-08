package cmd

import (
	"github.com/DougTea/rest-go-generator/pkg/gin"
	"golang.org/x/tools/go/packages"
	"k8s.io/code-generator/pkg/util"
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
	if len(arguments.InputDirs) == 0 {
		cpkg := util.CurrentPackage()
		p, err := packages.Load(nil, ".")
		if err != nil && len(p) != 1 {
			klog.Fatalf("Error: cannot determine input package.Detail: %v", err)
		}
		arguments.InputDirs = append(arguments.InputDirs, p[0].PkgPath)
		arguments.InputDirs = append(arguments.InputDirs, cpkg)
	}

	// Custom args.
	// customArgs := &gin.CustomArgs{}
	// pflag.CommandLine.StringSliceVar(&customArgs.BoundingDirs, "bounding-dirs", customArgs.BoundingDirs,
	// "Comma-separated list of import paths which bound the types for which deep-copies will be generated.")
	// arguments.CustomArgs = customArgs

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
