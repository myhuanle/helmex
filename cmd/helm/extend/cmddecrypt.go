package extend

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type DecryptCmdOptions struct {
	// RSAPrivateKeyPath RSA 私钥在本地的路径;
	// +Required;
	RSAPrivateKeyPath string
	// InputFile 要解密的内容的本地文件路径, 如果是 "-" 表示从 stdin 读取;
	// +Required;
	InputFile string
	// OutputFile 解密后的内容输出到文件中.
	// +Required;
	OutputFile string
}

func (o DecryptCmdOptions) Validate() error {
	if o.RSAPrivateKeyPath == "" {
		return errors.New("RSAPrivateKeyPath cannot be empty")
	}
	if _, err := os.Stat(o.RSAPrivateKeyPath); err != nil {
		return fmt.Errorf("failed to stat RSAPrivateKeyPath, %w", err)
	}
	if o.InputFile == "" {
		return errors.New("InputFile is required")
	}
	if o.InputFile != "-" {
		if _, err := os.Stat(o.InputFile); err != nil {
			return fmt.Errorf("failed to stat InputFile, %w", err)
		}
	}
	if o.OutputFile == "" {
		return errors.New("OutputFile cannot be empty")
	}
	return nil
}

func RunDecrypt(options *DecryptCmdOptions, out io.Writer) error {
	// read raw content;
	var data []byte
	var err error
	if options.InputFile == "-" {
		if data, err = io.ReadAll(os.Stdin); err != nil {
			return fmt.Errorf("failed to read data from stdin, %w", err)
		}
	} else {
		if data, err = os.ReadFile(options.InputFile); err != nil {
			return fmt.Errorf("failed to read data from input file %s, %w", options.InputFile, err)
		}
	}

	// decrypt data;
	secrettool, err := NewSecretDecoder(options.RSAPrivateKeyPath)
	if err != nil {
		return err
	}
	rawData, err := secrettool.Decode(string(data))
	if err != nil {
		return fmt.Errorf("failed to decrypt data, %w", err)
	}

	// create file to write
	absOutputFile, err := filepath.Abs(options.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to parse output file path to abs, %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(absOutputFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory to write decrypted data, %w", err)
	}
	if err = os.WriteFile(absOutputFile, rawData, 0644); err != nil {
		return fmt.Errorf("failed to write decrypted data into output file, %w", err)
	}
	return nil
}
