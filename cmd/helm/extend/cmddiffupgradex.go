package extend

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type DiffUpgradeXCmdOptions struct {
	// DataDir 临时数据目录;
	// +required;
	DataDir string
	// Manifest 清单文件;
	// +required;
	Manifest string
	// Services 指定只渲染特定的服务模板, 默认渲染全部;
	// +optional;
	Services []string
}

func (o DiffUpgradeXCmdOptions) Validate() error {
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

func RunDiffUpgradeX(options *DiffUpgradeXCmdOptions, out io.Writer) error {
	if err := options.Validate(); err != nil {
		return fmt.Errorf("failed to validate input options, %w", err)
	}

	manifest, err := loadManifest(options.Manifest)
	if err != nil {
		return fmt.Errorf("failed to load manifest, %w", err)
	}

	// 圈定目标服务, 如果未指定特定的服务, 则渲染所有的服务;
	services := options.Services
	if len(services) == 0 {
		for _, serviceList := range manifest.sortedServices {
			for _, s := range serviceList {
				services = append(services, s.Name)
			}
		}
	}

	// 执行 diff upgrade;
	for _, serviceName := range services {
		helmReleaseName := releaseName(manifest.K8s, manifest.Namespace, serviceName)
		chartDir := serviceChartDir(options.DataDir, serviceName)
		args := []string{"diff upgrade", helmReleaseName, chartDir, "-f", filepath.Join(chartDir, "values.yaml")}
		c := exec.Command(os.Args[0], args...)
		c.Stderr = out
		c.Stdout = out
		if err := c.Run(); err != nil {
			return err
		}
	}
	return nil
}
