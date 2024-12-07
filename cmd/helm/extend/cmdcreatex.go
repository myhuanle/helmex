package extend

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CreateXCmdOptions struct {
	ChartName       string
	ChartFolderName string
	ChartVersion    string
	AppVersion      string
	NoCharts        bool
	NoTemplates     bool
	NoValues        bool
}

func (o CreateXCmdOptions) Validate() error {
	if o.ChartName == "" {
		return errors.New("ChartName cannot be empty")
	}
	if o.ChartVersion == "" {
		return errors.New("ChartVersion cannot be empty")
	}
	if o.AppVersion == "" {
		return errors.New("AppVersion cannot be empty")
	}
	return nil
}

func RunCreateX(options *CreateXCmdOptions, out io.Writer) error {
	if err := options.Validate(); err != nil {
		return fmt.Errorf("failed to validate input options, %w", err)
	}

	chartMetaFile := filepath.Join("./", options.ChartName, "Chart.yaml")

	// default helm create;
	c := exec.Command(os.Args[0], "create", options.ChartName)
	b, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run `%s create %s`, err: %s", os.Args[0], options.ChartName, string(b))
	}

	// clear unused content;
	if options.NoValues {
		valuesFile := filepath.Join("./", options.ChartName, "values.yaml")
		if err = os.Remove(valuesFile); err != nil {
			return fmt.Errorf("failed to remove values.yaml from initial chart, %w", err)
		}
	}
	if options.NoTemplates {
		templatesDir := filepath.Join("./", options.ChartName, "templates")
		if err = os.RemoveAll(templatesDir); err != nil {
			return fmt.Errorf("failed to remove templates directory from initial chart, %w", err)
		}
		if err = os.MkdirAll(templatesDir, 0755); err != nil {
			return fmt.Errorf("failed to create empty templates directory for initial chart, %w", err)
		}
	}
	if options.NoCharts {
		chartsDir := filepath.Join("./", options.ChartName, "charts")
		if err = os.RemoveAll(chartsDir); err != nil {
			return fmt.Errorf("failed to remove charts directory from initial chart, %w", err)
		}
	}

	// replace chart version;
	b, err = os.ReadFile(chartMetaFile)
	if err != nil {
		return fmt.Errorf("failed to read chart meta file from `%s`, %w", chartMetaFile, err)
	}
	buff := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "version:") {
			line = fmt.Sprintf("version: %s", options.ChartVersion)
		}
		if buff.Len() > 0 {
			buff.WriteString("\n")
		}
		buff.WriteString(line)
	}
	if err = os.WriteFile(chartMetaFile, buff.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to rewrite chart version, %w", err)
	}

	// replace app version;
	b, err = os.ReadFile(chartMetaFile)
	if err != nil {
		return fmt.Errorf("failed to read chart meta file from `%s`, %w", chartMetaFile, err)
	}
	buff.Reset()
	scanner = bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "appVersion:") {
			line = fmt.Sprintf("appVersion: %s", options.AppVersion)
		}
		if buff.Len() > 0 {
			buff.WriteString("\n")
		}
		buff.WriteString(line)
	}
	if err = os.WriteFile(chartMetaFile, buff.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to rewrite app version, %w", err)
	}

	// folder rename;
	if options.ChartFolderName != "" {
		if err = os.Rename(options.ChartName, options.ChartFolderName); err != nil {
			return fmt.Errorf("failed to rename chart folder from %s to %s, %w", options.ChartName, options.ChartFolderName)
		}
	}
	return nil
}
