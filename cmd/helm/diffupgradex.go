// BY ZWF;
package main

import (
	"io"

	"helm.sh/helm/v3/cmd/helm/extend"

	"github.com/spf13/cobra"
)

func newDiffUpgradeXCmd(out io.Writer) *cobra.Command {
	options := &extend.DiffUpgradeXCmdOptions{}
	cmd := &cobra.Command{
		Use:   "diffupgradex",
		Short: "diff upgrade the helm chart",
		RunE: func(_ *cobra.Command, args []string) error {
			return extend.RunDiffUpgradeX(options, out)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&options.DataDir, "data-dir", "d", "./", "set the temporary data directory")
	f.StringVarP(&options.Manifest, "file", "f", "", "set manifest file")
	f.StringSliceVarP(&options.Services, "service", "s", []string{}, "set services to render template")

	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	return cmd
}
