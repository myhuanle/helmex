// BY ZWF;
package main

import (
	"fmt"
	"io"
	"os"

	"helm.sh/helm/v3/cmd/helm/extend"

	"github.com/spf13/cobra"
)

func newCreateXCmd(out io.Writer) *cobra.Command {
	options := &extend.CreateXCmdOptions{}
	cmd := &cobra.Command{
		Use:     "createx",
		Short:   "create helm chart scafford",
		Example: fmt.Sprintf(`%s createx --chart-name=xx --chart-foldername=0.12.11-0.4.1 --chart-version=0.12.11-0.4.1 --app-version=0.12.11`, os.Args[0]),
		RunE: func(_ *cobra.Command, args []string) error {
			return extend.RunCreateX(options, out)
		},
	}

	f := cmd.Flags()
	f.StringVar(&options.ChartName, "chart-name", "", "required to set the chart name")
	f.StringVar(&options.ChartFolderName, "chart-foldername", "", "optional to set the chart folder name")
	f.StringVar(&options.ChartVersion, "chart-version", "", "required to set the chart version")
	f.StringVar(&options.AppVersion, "app-version", "", "required to set the app version")
	f.BoolVar(&options.NoValues, "no-values", true, "optional, if true no values.yaml in generated chart directory")
	f.BoolVar(&options.NoCharts, "no-charts", true, "optional, if true no charts directory in generated chart directory")
	f.BoolVar(&options.NoTemplates, "no-templates", true, "optional, if true no templates directory content in generated chart directory")

	if err := cmd.MarkFlagRequired("chart-name"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("chart-version"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("app-version"); err != nil {
		panic(err)
	}

	return cmd
}
