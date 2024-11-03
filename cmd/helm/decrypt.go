// BY ZWF;
package main

import (
	"io"

	"helm.sh/helm/v3/cmd/helm/extend"

	"github.com/spf13/cobra"
)

func newDecryptCmd(out io.Writer) *cobra.Command {
	options := &extend.DecryptCmdOptions{}
	cmd := &cobra.Command{
		Use:   "decrypt",
		Short: "decrypt data with RSA private key",
		RunE: func(_ *cobra.Command, args []string) error {
			return extend.RunDecrypt(options, out)
		},
	}

	f := cmd.Flags()
	f.StringVar(&options.RSAPrivateKeyPath, "private-key", "", "set the private key file path")
	f.StringVarP(&options.InputFile, "input", "i", "-", "set the input file path, if it equals '-', then will read data from stdin")
	f.StringVarP(&options.OutputFile, "output", "o", "./decrypted.dat", "set the output file path")

	if err := cmd.MarkFlagRequired("private-key"); err != nil {
		panic(err)
	}

	return cmd
}
