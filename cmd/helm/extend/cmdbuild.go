package extend

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type BuildCmdOptions struct {
	// DataDir 临时数据目录;
	// +required;
	DataDir string
	// Manifest 清单文件;
	// +required;
	Manifest string
}

func (o BuildCmdOptions) Validate() error {
	if o.DataDir == "" {
		return errors.New("dataDir cannot be empty")
	}
	if _, err := os.Stat(o.DataDir); err != nil {
		return fmt.Errorf("failed to stat dataDir, %w", err)
	}

	if o.Manifest == "" {
		return errors.New("manifest cannot be empty")
	}
	if _, err := os.Stat(o.Manifest); err != nil {
		return fmt.Errorf("failed to stat manifest file, %w", err)
	}
	return nil
}

func RunBuild(options *BuildCmdOptions, out io.Writer) error {
	if err := options.Validate(); err != nil {
		return fmt.Errorf("failed to validate input options, %w", err)
	}

	manifest, err := loadManifest(options.Manifest)
	if err != nil {
		return fmt.Errorf("failed to load manifest, %w", err)
	}

	if err = buildChartFrom(options, manifest, out); err != nil {
		return fmt.Errorf("failed to build chart from manifest, %w", err)
	}
	return nil
}
