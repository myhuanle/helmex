package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type applyCmdOptions struct {
	// DataDir 临时数据目录;
	DataDir string
	// Manifest 清单文件;
	Manifest string
}

func newApplyCmd(out io.Writer) *cobra.Command {
	options := &applyCmdOptions{}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "apply manifest",
		RunE: func(_ *cobra.Command, args []string) error {
			return runApply(options, out)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&options.DataDir, "data-dir", "d", "./", "set the temporary data directory")
	f.StringVarP(&options.Manifest, "file", "f", "", "set manifest file")

	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}

	return cmd
}

func runApply(options *applyCmdOptions, out io.Writer) error {
	//bin := os.Args[0]
	//b, err := exec.Command(bin, "upgrade", "-h").CombinedOutput()
	//if err != nil {
	//	return err
	//}

	manifest, err := loadManifest(options.Manifest)
	if err != nil {
		return fmt.Errorf("failed to load manifest, %w", err)
	}

	if err = buildChartFrom(options, manifest, out); err != nil {
		return fmt.Errorf("failed to build chart from manifest, %w", err)
	}
	return nil
}

func loadManifest(file string) (*Manifest, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file, %w", err)
	}

	m := &Manifest{}
	if err = yaml.Unmarshal(b, m); err != nil {
		return nil, fmt.Errorf("failed to decode manifest from yaml, %w", err)
	}
	if err = m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest, %w", err)
	}

	// 按优先级将 service 排序;
	serviceNameSet := make(map[string]struct{})
	servicePriorityToList := make(map[int][]*Service)
	servicePriorityList := make([]int, 0)
	for _, service := range m.Services {
		if _, ok := serviceNameSet[service.Name]; ok {
			return nil, fmt.Errorf("duplicated service name %s", service.Name)
		}
		serviceList := servicePriorityToList[service.Priority]
		if serviceList == nil {
			serviceList = make([]*Service, 0)
		}
		servicePriorityToList[service.Priority] = append(serviceList, service)
		serviceNameSet[service.Name] = struct{}{}
		servicePriorityList = append(servicePriorityList, service.Priority)
	}
	sort.Ints(servicePriorityList)

	sortedServices := make([][]*Service, 0, len(servicePriorityList))
	for _, priority := range servicePriorityList {
		serviceList := servicePriorityToList[priority]
		sortedServices = append(sortedServices, serviceList)
	}

	m.sortedServices = sortedServices

	return m, nil
}

func buildChartFrom(options *applyCmdOptions, m *Manifest, _ io.Writer) error {
	chartsDir := filepath.Join(options.DataDir, "charts")
	if err := os.RemoveAll(chartsDir); err != nil {
		return fmt.Errorf("failed to clear charts directory, %w", err)
	}
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return fmt.Errorf("failed to create charts directory, %w", err)
	}

	// download all from remote;
	for _, service := range m.Services {
		//fmt.Fprintf(logWriter, "[I] cloning service chart %s ...\n", service.Name)
		if err := gitClone(
			service.Template.GitRef.URL,
			service.Template.GitRef.Branch,
			service.Template.GitRef.Commit,
			filepath.Join(chartsDir, service.Name),
			service.Template.GitRef.SSHKeyPath,
		); err != nil {
			return fmt.Errorf("failed to git clone service %s from %s, %w", service.Name, service.Template.GitRef.URL, err)
		}
	}
	return nil
}

func gitClone(url, branch, commit, toDir, sshKeyPath string) error {
	if err := os.MkdirAll(toDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory, %w", err)
	}

	if sshKeyPath == "" {
		sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}
	auth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if err != nil {
		fmt.Printf("Error creating SSH public keys: %v\n", err)
		os.Exit(1)
	}

	repo, err := git.PlainClone(toDir, false, &git.CloneOptions{
		URL:           url,
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		return err
	}

	// checkout to commit;
	if commit != "" {
		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree, %w", err)
		}

		if err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(commit),
		}); err != nil {
			return fmt.Errorf("failed to checkout to %s, %w", commit, err)
		}
	}
	return nil
}

type Manifest struct {
	K8s       string     `yaml:"k8s"`
	Namespace string     `yaml:"namespace"`
	Services  []*Service `yaml:"services"`

	// internal use only;
	sortedServices [][]*Service
}

func (m Manifest) Validate() error {
	if m.K8s == "" {
		return errors.New("k8s cannot be empty")
	}
	if m.Namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	for idx, service := range m.Services {
		if err := service.Validate(fmt.Sprintf("services[%d]", idx)); err != nil {
			return err
		}
	}
	return nil
}

type Service struct {
	Name     string          `yaml:"name"`
	Priority int             `yaml:"priority"`
	Template ServiceTemplate `yaml:"template"`
	Value    ServiceValue    `yaml:"value"`
}

func (s Service) Validate(fieldPath string) error {
	if s.Name == "" {
		return fmt.Errorf("%s.name cannot be empty", fieldPath)
	}
	if err := s.Template.Validate(fmt.Sprintf("%s.template", fieldPath)); err != nil {
		return err
	}
	if err := s.Value.Validate(fmt.Sprintf("%s.value", fieldPath)); err != nil {
		return err
	}
	return nil
}

type ServiceTemplate struct {
	GitRef *GitRef `yaml:"gitRef"`
}

func (t ServiceTemplate) Validate(fieldPath string) error {
	if t.GitRef == nil {
		return fmt.Errorf("%s.gitRef is required", fieldPath)
	}
	if err := t.GitRef.Validate(fmt.Sprintf("%s.gitRef", fieldPath)); err != nil {
		return err
	}
	return nil
}

type ServiceValue struct {
	GitRef *GitRef `yaml:"gitRef"`
}

func (t ServiceValue) Validate(fieldPath string) error {
	if t.GitRef == nil {
		return fmt.Errorf("%s.gitRef is required", fieldPath)
	}
	if err := t.GitRef.Validate(fmt.Sprintf("%s.gitRef", fieldPath)); err != nil {
		return err
	}
	return nil
}

type GitRef struct {
	// URL;
	// +Required;
	URL string `yaml:"url"`
	// Branch;
	// +Required;
	Branch string `yaml:"branch"`
	// Commit;
	// +Optional;
	Commit string `yaml:"commit"`
	// Path;
	// +Required;
	Path string `yaml:"path"`
	// SSHKeyPath for cloning code;
	// +Optional;
	SSHKeyPath string `yaml:"sshKeyPath"`
}

func (r GitRef) Validate(fieldPath string) error {
	if r.URL == "" {
		return fmt.Errorf("%s.url cannot be empty", fieldPath)
	}
	if r.Branch == "" {
		return fmt.Errorf("%s.branch cannot be empty", fieldPath)
	}
	if r.Path == "" {
		return fmt.Errorf("%s.path cannot be empty", fieldPath)
	}
	if r.SSHKeyPath != "" {
		if _, err := os.Stat(r.SSHKeyPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("invalid %s.sshKeyPath, file does not exist", r.SSHKeyPath)
			}
			return fmt.Errorf("invalid %s.sshKeyPath, could not stat it, %w", fieldPath, err)
		}
	}
	return nil
}
