package extend

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	// Context 设置显示 diff 的上下文行数;
	// +optional;
	// 默认为 -1, 显示全部;
	Context int
	// AllowUnReleased enables diffing of releases that are not yet deployed via Helm;
	// +optional;
	// 默认为 false;
	AllowUnReleased bool
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

	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", manifest.K8s+".kubeconfig")

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
		errBuff := bytes.NewBuffer(nil)
		helmReleaseName := releaseName(manifest.K8s, manifest.Namespace, serviceName)
		chartDir := serviceChartDir(options.DataDir, serviceName)
		args := []string{
			"--kubeconfig", kubeconfig,
			"-n", manifest.Namespace,
			"diff", "upgrade", helmReleaseName, chartDir,
			"-f", filepath.Join(chartDir, "values.yaml"),
			"--set", fmt.Sprintf("k8sName=%s", manifest.K8s),
			"--set", fmt.Sprintf("namespace=%s", manifest.Namespace),
			"--context", strconv.Itoa(options.Context),
			fmt.Sprintf("--allow-unreleased=%t", options.AllowUnReleased),
		}
		c := exec.Command(os.Args[0], args...)
		c.Stderr = errBuff
		c.Stdout = out
		if err := c.Run(); err != nil {
			return fmt.Errorf("failed to diff upgrade service, %s", errBuff.String())
		}
	}
	return nil
}
