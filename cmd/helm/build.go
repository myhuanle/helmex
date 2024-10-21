package main

import (
	"io"

	"helm.sh/helm/v3/cmd/helm/extend"

	"github.com/spf13/cobra"
)

func newBuildCmd(out io.Writer) *cobra.Command {
	options := &extend.BuildCmdOptions{}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build manifest",
		RunE: func(_ *cobra.Command, args []string) error {
			return extend.RunBuild(options, out)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&options.DataDir, "data-dir", "d", "./", "set the temporary data directory")
	f.StringVarP(&options.Manifest, "file", "f", "", "set manifest file")

	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	return cmd
}
