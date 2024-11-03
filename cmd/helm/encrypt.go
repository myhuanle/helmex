// BY ZWF;
package main

import (
	"io"

	"helm.sh/helm/v3/cmd/helm/extend"

	"github.com/spf13/cobra"
)

func newEncryptCmd(out io.Writer) *cobra.Command {
	options := &extend.EncryptCmdOptions{}
	cmd := &cobra.Command{
		Use:   "encrypt",
		Short: "encrypt data with RSA public key",
		RunE: func(_ *cobra.Command, args []string) error {
			return extend.RunEncrypt(options, out)
		},
	}

	f := cmd.Flags()
	f.StringVar(&options.RSAPublicKeyPath, "public-key", "", "set the public key file path")
	f.StringVarP(&options.InputFile, "input", "i", "-", "set the input file path, if it equals '-', then will read data from stdin")
	f.StringVarP(&options.OutputFile, "output", "o", "", "set the output file path, if it's empty, then will output to stdout")

	if err := cmd.MarkFlagRequired("public-key"); err != nil {
		panic(err)
	}

	return cmd
}
