package extend

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	K8s       string `yaml:"k8s"`
	Namespace string `yaml:"namespace"`
	// Labels 标签数据;
	// 必须包含 k8sName,
	// 必须包含 env, 表示环境分类, enum: ["test", "stress", "preview", "prod"]
	// 必须包含 namespaceName, 表示命名空间描述;
	Labels   map[string]any `yaml:"labels"`
	Services []*Service     `yaml:"services"`

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
	if len(m.Labels) == 0 {
		return errors.New("labels cannot be empty")
	}

	k8sNameAny, ok := m.Labels["k8sName"]
	if !ok {
		return errors.New("labels.k8sName is required")
	}
	k8sName, ok := k8sNameAny.(string)
	if !ok {
		return errors.New("invalid labels.k8sName type, expected string")
	}
	if k8sName == "" {
		return errors.New("labels.k8sName cannot be empty")
	}
	if utf8.RuneCountInString(k8sName) > 50 {
		return errors.New("labels.k8sName is too long, it exceedes 50 UTF-8 characters")
	}

	envAny, ok := m.Labels["env"]
	if !ok {
		return errors.New("labels.env is required")
	}
	env, ok := envAny.(string)
	if !ok {
		return errors.New("invalid labels.env type, expected string")
	}
	if env != "test" && env != "stress" && env != "preview" && env != "prod" {
		return errors.New(`labels.env must be one of ["test", "stress", "preview", "prod"]`)
	}

	namespaceNameAny, ok := m.Labels["namespaceName"]
	if !ok {
		return errors.New("labels.namespaceName is required")
	}
	namespaceName, ok := namespaceNameAny.(string)
	if !ok {
		return errors.New("invalid labels.namespaceName type, expected string")
	}
	if namespaceName == "" {
		return errors.New("labels.namespaceName cannot be empty")
	}
	if utf8.RuneCountInString(namespaceName) > 50 {
		return errors.New("labels.namespaceName is too long, it exceedes 50 UTF-8 characters")
	}

	for idx, service := range m.Services {
		if err := service.Validate(fmt.Sprintf("services[%d]", idx)); err != nil {
			return err
		}
	}
	return nil
}

func (m Manifest) Check(out io.Writer) (pass bool) {
	pass = true
	for _, service := range m.Services {
		// 检测value文件的引用路径是否与集群名一致;
		valuePath := service.Value.GitRef.Path
		if !strings.Contains(valuePath, m.K8s) {
			fmt.Fprintf(out, "\033[33mWarning: value path `%s` of service `%s` was not matched with k8s name `%s`\033[0m\n", valuePath, service.Name, m.K8s)
			pass = false
		}
	}
	return pass
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
	// SecretValueDecryptKeyPath 密文解密私钥;
	// +optional;
	SecretValueDecryptKeyPath string `yaml:"secretValueDecryptKeyPath"`
}

func (t ServiceValue) Validate(fieldPath string) error {
	if t.GitRef == nil {
		return fmt.Errorf("%s.gitRef is required", fieldPath)
	}
	if err := t.GitRef.Validate(fmt.Sprintf("%s.gitRef", fieldPath)); err != nil {
		return err
	}
	if t.SecretValueDecryptKeyPath != "" {
		p, err := fileAbsPath(t.SecretValueDecryptKeyPath)
		if err != nil {
			return fmt.Errorf("invalid %s.secretValueDecryptKeyPath value, could not parse it to abstract path, %w", fieldPath, err)
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("invalid %s.secretValueDecryptKeyPath value, %w", fieldPath, err)
		}
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
			servicePriorityList = append(servicePriorityList, service.Priority)
		}
		servicePriorityToList[service.Priority] = append(serviceList, service)
		serviceNameSet[service.Name] = struct{}{}
	}
	sort.Ints(servicePriorityList)

	sortedServices := make([][]*Service, 0, len(servicePriorityList))
	for i := len(servicePriorityList) - 1; i >= 0; i-- {
		priority := servicePriorityList[i]
		serviceList := servicePriorityToList[priority]
		sortedServices = append(sortedServices, serviceList)
	}

	m.sortedServices = sortedServices

	return m, nil
}

func buildChartFrom(options *BuildCmdOptions, m *Manifest, _ io.Writer) error {
	// download all from remote;
	gitCacheDir := filepath.Join(options.DataDir, "cache", "git")
	if err := os.RemoveAll(gitCacheDir); err != nil {
		return fmt.Errorf("failed to clear git cache directory, %w", err)
	}
	if err := os.MkdirAll(gitCacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create git cache directory, %w", err)
	}
	// clear chart directory;
	if err := os.RemoveAll(chartDir(options.DataDir)); err != nil {
		return fmt.Errorf("failed to clear chart directory, %w", err)
	}
	for _, service := range m.Services {
		serviceChartDir := serviceChartDir(options.DataDir, service.Name)
		// copy templates;
		cacheDir := filepath.Join(gitCacheDir, gitRefHash(service.Template.GitRef))
		if _, err := os.Stat(filepath.Join(cacheDir, ".git")); err != nil {
			// pull from the remote if the cache does not exist;
			if err := gitCloneWithBin(service.Template.GitRef.URL, service.Template.GitRef.Branch, service.Template.GitRef.Commit, cacheDir, service.Template.GitRef.SSHKeyPath); err != nil {
				return fmt.Errorf("failed to git clone service %s's template from %s, %w", service.Name, service.Template.GitRef.URL, err)
			}
		}
		copyFromPath := filepath.Join(cacheDir, service.Template.GitRef.Path)
		if err := copyTemplates(copyFromPath, serviceChartDir); err != nil {
			return fmt.Errorf("failed to copy chart template for service %s, %w", service.Name, err)
		}

		// copy values;
		cacheDir = filepath.Join(gitCacheDir, gitRefHash(service.Value.GitRef))
		if _, err := os.Stat(filepath.Join(cacheDir, ".git")); err != nil {
			// pull from the remote if the cache does not exist;
			if err := gitCloneWithBin(service.Value.GitRef.URL, service.Value.GitRef.Branch, service.Value.GitRef.Commit, cacheDir, service.Value.GitRef.SSHKeyPath); err != nil {
				return fmt.Errorf("failed to git clone service %s's value from %s, %w", service.Name, service.Template.GitRef.URL, err)
			}
		}
		copyFromPath = filepath.Join(cacheDir, service.Value.GitRef.Path)
		if err := copyValues(copyFromPath, serviceChartDir, secretDecoderFromFile(service.Value.SecretValueDecryptKeyPath)); err != nil {
			return fmt.Errorf("failed to copy chart values for service %s, %w", service.Name, err)
		}
	}

	return nil
}

// func gitClone(url, branch, commit, toDir, sshKeyPath string) error {
// 	if err := os.RemoveAll(toDir); err != nil {
// 		return fmt.Errorf("failed to clear directory, %w", err)
// 	}
// 	if err := os.MkdirAll(toDir, 0755); err != nil {
// 		return fmt.Errorf("failed to create directory, %w", err)
// 	}

// 	// read ssh key;
// 	if sshKeyPath == "" {
// 		sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
// 	}
// 	publicKeys, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
// 	if err != nil {
// 		return err
// 	}
// 	publicKeys.HostKeyCallback = gossh.InsecureIgnoreHostKey()

// 	// git clone;
// 	repo, err := git.PlainClone(toDir, false, &git.CloneOptions{
// 		URL:           url,
// 		Auth:          publicKeys,
// 		ReferenceName: plumbing.NewBranchReferenceName(branch),
// 		SingleBranch:  true,
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	// checkout to commit;
// 	if commit != "" {
// 		worktree, err := repo.Worktree()
// 		if err != nil {
// 			return fmt.Errorf("failed to get worktree, %w", err)
// 		}

// 		if err = worktree.Checkout(&git.CheckoutOptions{
// 			Hash: plumbing.NewHash(commit),
// 		}); err != nil {
// 			return fmt.Errorf("failed to checkout to %s, %w", commit, err)
// 		}
// 	}
// 	return nil
// }

func gitCloneWithBin(url, branch, commit, toDir, sshKeyPath string) error {
	if err := os.RemoveAll(toDir); err != nil {
		return fmt.Errorf("failed to clear directory, %w", err)
	}
	if err := os.MkdirAll(toDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory, %w", err)
	}

	// read ssh key;
	if sshKeyPath == "" {
		sshKeyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}
	if err := os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s", sshKeyPath)); err != nil {
		return fmt.Errorf("failed to set GIT_SSH_COMMAND env, %w", err)
	}

	// git clone target;
	c := exec.Command("git", "clone", "--single-branch", "--branch", branch, url, "./")
	c.Dir = toDir
	b, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to git clone %s, err: %s", url, string(b))
	}
	// checkout to commit;
	if commit != "" {
		c := exec.Command("git", "checkout", commit)
		c.Dir = toDir
		b, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to git checkout to commit %s in %s branch %s, err: %s", commit, url, branch, string(b))
		}
	}
	return nil
}

func chartDir(rootDataDir string) string {
	return filepath.Join(rootDataDir, "charts")
}

func serviceChartDir(rootDataDir, serviceName string) string {
	return filepath.Join(chartDir(rootDataDir), serviceName)
}

func copyTemplates(fromDir, toDir string) error {
	if err := os.RemoveAll(toDir); err != nil {
		return fmt.Errorf("failed to clear target directory, %w", err)
	}
	if err := os.MkdirAll(toDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory, %w", err)
	}

	pathInfo, err := os.Stat(fromDir)
	if err != nil {
		return fmt.Errorf("failed to stat %s, %w", fromDir, err)
	}
	if pathInfo.IsDir() {
		fromDir = fromDir + "/."
	}
	msg, err := exec.Command("cp", "-r", fromDir, toDir).CombinedOutput()
	if err != nil {
		return errors.New(string(msg))
	}
	return nil
}

func copyValues(fromDir, toDir string, secretDecoder SecretDecoder) error {
	valuesBuff := bytes.NewBuffer(nil)
	// 扫描目录中所有的 values 文件, 尝试对密码解密;
	valuesEntries, err := os.ReadDir(fromDir)
	if err != nil {
		return fmt.Errorf("failed to scan directory to find value files, %w", err)
	}
	for _, valuesEntry := range valuesEntries {
		if valuesEntry.IsDir() {
			continue
		}
		if ext := filepath.Ext(valuesEntry.Name()); ext != ".yml" && ext != ".yaml" {
			continue
		}
		fromValueFile := filepath.Join(fromDir, valuesEntry.Name())
		b, err := os.ReadFile(fromValueFile)
		if err != nil {
			return fmt.Errorf("failed to read value file content, %w", err)
		}
		if valuesBuff.Len() > 0 {
			valuesBuff.WriteString("\n\n\n")
		}
		fmt.Fprintf(valuesBuff, "# -------------------- FROM %s ------------------ #", valuesEntry.Name())
		scanner := bufio.NewScanner(bytes.NewReader(b))
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			lineNum++
			if strings.Contains(line, "helmsecret://") {
				if secretDecoder == nil {
					return fmt.Errorf("we detect helmsecret:// in values file %s:%d, but no secretValueDecryptKeyFile was not provided!", fromValueFile, lineNum)
				}
				// decode secret value;
				colonIdx := strings.Index(line, ":")
				key := line[0:colonIdx]
				secretValue := strings.TrimSpace(line[colonIdx+1:])
				plainValue, err := secretDecoder.Decode(secretValue)
				if err != nil {
					return fmt.Errorf("failed to decode secret at %s:%d, error: %w", fromValueFile, lineNum, err)
				}
				line = fmt.Sprintf("%s: %s", key, plainValue)
			}
			if valuesBuff.Len() > 0 {
				valuesBuff.WriteString("\n")
			}
			valuesBuff.WriteString(line)
		}
	}

	toValuesFile := filepath.Join(toDir, "values.yaml")
	if err = os.WriteFile(toValuesFile, valuesBuff.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write values content to dest file, %w", err)
	}
	return nil
}

func gitRefHash(gitRef *GitRef) string {
	buff := bytes.NewBuffer(nil)
	buff.WriteString(gitRef.URL)
	buff.WriteString(gitRef.Branch)
	buff.WriteString(gitRef.Commit)
	return fmt.Sprintf("%x", md5.Sum(buff.Bytes()))
}
