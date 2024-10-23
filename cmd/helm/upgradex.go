// BY ZWF;
package main

import (
	"io"

	"helm.sh/helm/v3/cmd/helm/extend"

	"github.com/spf13/cobra"
)

func newUpgradeXCmd(out io.Writer) *cobra.Command {
	options := &extend.UpgradeXCmdOptions{}
	cmd := &cobra.Command{
		Use:   "upgradex",
		Short: "upgrade the helm chart",
		RunE: func(_ *cobra.Command, args []string) error {
			return extend.RunUpgradeX(options, out)
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
