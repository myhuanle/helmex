package extend

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

type SecretEncoder interface {
	Encode(data []byte) (string, error)
}

type SecretDecoder interface {
	Decode(secretData string) ([]byte, error)
}

var _ SecretEncoder = (*SecretTool)(nil)
var _ SecretDecoder = (*SecretTool)(nil)

type SecretTool struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func NewSecretEncoder(publicKeyFile string) (*SecretTool, error) {
	b, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key from file, %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("bad public key, could not decode it as pem")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("bad public key, %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid rsa public key")
	}
	return &SecretTool{
		publicKey:  rsaPub,
		privateKey: nil,
	}, nil
}

func NewSecretDecoder(privateKeyFile string) (*SecretTool, error) {
	b, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from file, %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("bad private key, could not decode it as pem")
	}

	var privateKey *rsa.PrivateKey
	var ok bool
	// 尝试PKCS8格式解析
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if privateKey, ok = key.(*rsa.PrivateKey); !ok {
			return nil, fmt.Errorf("invalid RSA private key, %w", err)
		}
	}
	// 如果PKCS8解析失败，尝试PKCS1格式
	if privateKey == nil {
		if privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
			return nil, fmt.Errorf("invalid RSA private key, %w", err)
		}
	}
	if privateKey == nil {
		return nil, errors.New("could not parse private key with PKCS8 and PKCS1")
	}
	return &SecretTool{
		publicKey:  nil,
		privateKey: privateKey,
	}, nil
}

func (t *SecretTool) Encode(data []byte) (string, error) {
	if t.publicKey == nil {
		return "", errors.New("secret tool had not been inited as encoder")
	}
	encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, t.publicKey, data)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("helmsecret://%s", base64.StdEncoding.EncodeToString(encryptedData)), nil
}

func (t *SecretTool) Decode(secretData string) ([]byte, error) {
	if t.privateKey == nil {
		return nil, errors.New("secret tool had not been inited as decoder")
	}
	if !strings.HasPrefix(secretData, "helmsecret://") {
		return nil, errors.New("invalid secret data, missing prefix helmsecret://")
	}
	secretData = strings.TrimPrefix(secretData, "helmsecret://")
	b, err := base64.StdEncoding.DecodeString(secretData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret content with base64, %w", err)
	}
	b, err = rsa.DecryptPKCS1v15(rand.Reader, t.privateKey, b)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret with private key, %w", err)
	}
	return b, nil
}

// secretDecoderFromFile 从文件构造秘文解码器;
// 如果 privateKeyFilePath 未指定, 将返回 nil SecretDecoder;
func secretDecoderFromFile(privateKeyFilePath string) SecretDecoder {
	if privateKeyFilePath == "" {
		return nil
	}
	secretDecoder, err := NewSecretDecoder(privateKeyFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to new secret decoder from file %s, %v", privateKeyFilePath, err))
	}
	return secretDecoder
}
