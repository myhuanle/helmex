package extend

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TemplateXCmdOptions struct {
	// DataDir 临时数据目录;
	DataDir string
	// Manifest 清单文件;
	Manifest string
	// Services 指定只渲染特定的服务模板, 默认渲染全部;
	Services []string
}

func (o TemplateXCmdOptions) Validate() error {
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

func RunTemplateX(options *TemplateXCmdOptions, out io.Writer) error {
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

	// 执行渲染;
	for _, serviceName := range services {
		chartDir := serviceChartDir(options.DataDir, serviceName)
		args := []string{"template", chartDir, "-f", filepath.Join(chartDir, "values.yaml")}
		b, err := exec.Command(os.Args[0], args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %s:\n%s", os.Args[0], strings.Join(args, " "), string(b))
		}
		out.Write(b)
	}
	return nil
}
