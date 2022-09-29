package gin

import (
	"github.com/spf13/pflag"
	"k8s.io/gengo/args"
)

type CustomArgs struct {
	BoundingDirs []string // Only deal with types rooted under these dirs.
}

func NewDefaults() (*args.GeneratorArgs, *CustomArgs) {
	return args.Default().WithoutDefaultFlagParsing(), new(CustomArgs)
}

func (ca *CustomArgs) AddFlags(fs *pflag.FlagSet) {
}
