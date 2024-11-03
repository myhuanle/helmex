package extend

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type EncryptCmdOptions struct {
	// RSAPublicKeyPath RSA 公钥在本地的路径;
	// +Required;
	RSAPublicKeyPath string
	// InputFile 要加密的内容的本地文件路径, 如果是 "-" 表示从 stdin 读取;
	// +Required;
	InputFile string
	// OutputFile 加密后的内容输出到文件中.
	// +Optional, 如果为空, 将直接输出到控制台;
	OutputFile string
}

func (o EncryptCmdOptions) Validate() error {
	if o.RSAPublicKeyPath == "" {
		return errors.New("RSAPublicKeyPath cannot be empty")
	}
	if _, err := os.Stat(o.RSAPublicKeyPath); err != nil {
		return fmt.Errorf("failed to stat RSAPublicKeyPath, %w", err)
	}
	if o.InputFile == "" {
		return errors.New("InputFile is required")
	}
	if o.InputFile != "-" {
		if _, err := os.Stat(o.InputFile); err != nil {
			return fmt.Errorf("failed to stat InputFile, %w", err)
		}
	}
	return nil
}

func RunEncrypt(options *EncryptCmdOptions, out io.Writer) error {
	var data []byte
	var err error
	// read raw content;
	if options.InputFile == "-" {
		if data, err = io.ReadAll(os.Stdin); err != nil {
			return fmt.Errorf("failed to read data from stdin, %w", err)
		}
	} else {
		if data, err = os.ReadFile(options.InputFile); err != nil {
			return fmt.Errorf("failed to read data from input file %s, %w", options.InputFile, err)
		}
	}

	// use public key to encrypt content,
	secrettool, err := NewSecretEncoder(options.RSAPublicKeyPath)
	if err != nil {
		return err
	}
	encryptedString, err := secrettool.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data, %w", err)
	}

	// create file to write
	if options.OutputFile == "" {
		fmt.Fprintln(os.Stdout, encryptedString)
		return nil
	}
	absOutputFile, err := filepath.Abs(options.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to parse output file path to abs, %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(absOutputFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory to write encrypted data, %w", err)
	}
	if err = os.WriteFile(absOutputFile, []byte(encryptedString), 0644); err != nil {
		return fmt.Errorf("failed to write encrypted data into output file, %w", err)
	}
	return nil
}
