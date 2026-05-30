package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bnema/hexcheck/internal/glob"
	"gopkg.in/yaml.v3"
)

type Role string

const (
	RoleCore       Role = "core"
	RoleUsecase    Role = "usecase"
	RolePorts      Role = "ports"
	RoleAdapter    Role = "adapter"
	RoleEntrypoint Role = "entrypoint"
	RoleIgnore     Role = "ignore"
)

type Severity string

const (
	SeverityOff   Severity = "off"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

type Config struct {
	Version       int                    `yaml:"version"`
	Components    map[string]Component   `yaml:"components"`
	Rules         map[string]Severity    `yaml:"rules"`
	ExternalTypes ExternalTypes          `yaml:"externalTypes"`
	Heuristics    Heuristics             `yaml:"heuristics"`
	Mocking       Mocking                `yaml:"mocking"`
	RuleSettings  map[string]RuleSetting `yaml:"ruleSettings"`
	Allow         []Allow                `yaml:"allow"`
	Root          string                 `yaml:"-"`
}

type Component struct {
	Role  Role     `yaml:"role"`
	Paths []string `yaml:"paths"`
}

type ExternalTypes struct {
	FrameworkPackages   []string `yaml:"frameworkPackages"`
	AdapterTypePackages []string `yaml:"adapterTypePackages"`
}

type Heuristics struct {
	BusinessLogicThreshold                *int     `yaml:"businessLogicThreshold"`
	BusinessLogicMinStrongSignals         *int     `yaml:"businessLogicMinStrongSignals"`
	BusinessLogicMinWeakSignals           *int     `yaml:"businessLogicMinWeakSignals"`
	BusinessLogicMaxFunctionNodes         *int     `yaml:"businessLogicMaxFunctionNodes"`
	BusinessLogicMaxDiagnosticsPerPackage *int     `yaml:"businessLogicMaxDiagnosticsPerPackage"`
	BusinessLogicMode                     string   `yaml:"businessLogicMode"`
	BusinessLogicMinConfidence            string   `yaml:"businessLogicMinConfidence"`
	BusinessKeywords                      []string `yaml:"businessKeywords"`
	ExcludeTestFiles                      *bool    `yaml:"excludeTestFiles"`
}

type Mocking struct {
	GeneratedMockPaths        []string `yaml:"generatedMockPaths"`
	GeneratedMockNamePatterns []string `yaml:"generatedMockNamePatterns"`
}

type RuleSetting struct {
	ExcludePaths []string `yaml:"excludePaths"`
}

type Allow struct {
	Rule   string `yaml:"rule"`
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
}

type Match struct {
	Name string
	Role Role
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if hasExplicitVersion(data) && cfg.Version == 0 {
		return nil, errors.New("version: unsupported value 0")
	}
	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	cfg.Root = root
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Default() *Config {
	cfg := &Config{Version: 1}
	cfg.applyDefaults()
	return cfg
}

func (c *Config) ApplyDefaults() {
	c.applyDefaults()
}

func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Rules == nil {
		c.Rules = map[string]Severity{}
	}
	for rule, sev := range DefaultRuleSeverities() {
		if _, ok := c.Rules[rule]; !ok {
			c.Rules[rule] = sev
		}
	}
	if c.Heuristics.BusinessLogicThreshold == nil {
		c.Heuristics.BusinessLogicThreshold = intPtr(8)
	}
	if c.Heuristics.BusinessLogicMinStrongSignals == nil {
		c.Heuristics.BusinessLogicMinStrongSignals = intPtr(2)
	}
	if c.Heuristics.BusinessLogicMinWeakSignals == nil {
		c.Heuristics.BusinessLogicMinWeakSignals = intPtr(2)
	}
	if c.Heuristics.BusinessLogicMaxFunctionNodes == nil {
		c.Heuristics.BusinessLogicMaxFunctionNodes = intPtr(2000)
	}
	if c.Heuristics.BusinessLogicMaxDiagnosticsPerPackage == nil {
		c.Heuristics.BusinessLogicMaxDiagnosticsPerPackage = intPtr(10)
	}
	if c.Heuristics.ExcludeTestFiles == nil {
		defaultExcludeTestFiles := true
		c.Heuristics.ExcludeTestFiles = &defaultExcludeTestFiles
	}
	if c.Heuristics.BusinessLogicMode == "" {
		c.Heuristics.BusinessLogicMode = "audit"
	}
	if c.Heuristics.BusinessLogicMinConfidence == "" {
		c.Heuristics.BusinessLogicMinConfidence = "medium"
	}
	if len(c.Heuristics.BusinessKeywords) == 0 {
		c.Heuristics.BusinessKeywords = []string{"Validate", "Authorize", "Compute", "Calculate", "Apply", "Transition", "Can", "Detect", "Migrate", "Resolve", "Profile", "Score", "Ranking", "Restore", "Purge", "Update", "Performance", "Selected"}
	}
	if len(c.Mocking.GeneratedMockNamePatterns) == 0 {
		c.Mocking.GeneratedMockNamePatterns = []string{"Mock{{Interface}}", "{{Interface}}Mock"}
	}
}

func intPtr(v int) *int { return &v }

func hasExplicitVersion(data []byte) bool {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil || len(node.Content) == 0 {
		return false
	}
	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "version" {
			return true
		}
	}
	return false
}

func DefaultRuleSeverities() map[string]Severity {
	return map[string]Severity{
		"no-adapter-imports-in-core":           SeverityError,
		"no-infra-imports-in-usecase":          SeverityError,
		"no-framework-types-in-core":           SeverityError,
		"no-infra-types-in-ports":              SeverityError,
		"no-adapter-to-adapter-imports":        SeverityWarn,
		"suspicious-business-logic-in-adapter": SeverityWarn,
		"no-local-fakes-for-ports":             SeverityWarn,
		"missing-generated-mock-for-port":      SeverityWarn,
		"prefer-generated-mocks":               SeverityWarn,
	}
}

func (c *Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("version: unsupported value %d", c.Version)
	}
	for name, component := range c.Components {
		if name == "" {
			return errors.New("components: empty component name")
		}
		switch component.Role {
		case RoleCore, RoleUsecase, RolePorts, RoleAdapter, RoleEntrypoint, RoleIgnore:
		default:
			return fmt.Errorf("components.%s.role: unsupported role %q", name, component.Role)
		}
		if len(component.Paths) == 0 {
			return fmt.Errorf("components.%s.paths: must not be empty", name)
		}
	}
	if c.Heuristics.BusinessLogicThreshold != nil && *c.Heuristics.BusinessLogicThreshold < 0 {
		return errors.New("heuristics.businessLogicThreshold: must be >= 0")
	}
	if c.Heuristics.BusinessLogicMinStrongSignals != nil && *c.Heuristics.BusinessLogicMinStrongSignals < 0 {
		return errors.New("heuristics.businessLogicMinStrongSignals: must be >= 0")
	}
	if c.Heuristics.BusinessLogicMinWeakSignals != nil && *c.Heuristics.BusinessLogicMinWeakSignals < 0 {
		return errors.New("heuristics.businessLogicMinWeakSignals: must be >= 0")
	}
	if c.Heuristics.BusinessLogicMaxFunctionNodes != nil && *c.Heuristics.BusinessLogicMaxFunctionNodes < 1 {
		return errors.New("heuristics.businessLogicMaxFunctionNodes: must be >= 1")
	}
	if c.Heuristics.BusinessLogicMaxDiagnosticsPerPackage != nil && *c.Heuristics.BusinessLogicMaxDiagnosticsPerPackage < 1 {
		return errors.New("heuristics.businessLogicMaxDiagnosticsPerPackage: must be >= 1")
	}
	switch c.Heuristics.BusinessLogicMode {
	case "ci", "audit":
	default:
		return errors.New("heuristics.businessLogicMode: must be ci or audit")
	}
	switch c.Heuristics.BusinessLogicMinConfidence {
	case "low", "medium", "high":
	default:
		return errors.New("heuristics.businessLogicMinConfidence: must be low, medium, or high")
	}
	for rule, severity := range c.Rules {
		switch severity {
		case SeverityOff, SeverityWarn, SeverityError:
		default:
			return fmt.Errorf("rules.%s: unsupported severity %q", rule, severity)
		}
	}
	for i, allow := range c.Allow {
		if allow.Rule == "" || allow.Path == "" || allow.Reason == "" {
			return fmt.Errorf("allow[%d]: rule, path, and reason are required", i)
		}
	}
	return nil
}

func (c *Config) Severity(rule string) Severity {
	if c == nil || c.Rules == nil {
		return SeverityOff
	}
	return c.Rules[rule]
}

func (c *Config) RuleEnabled(rule string) bool {
	return c.Severity(rule) != SeverityOff
}

func (c *Config) ComponentForPath(path string) (Match, bool) {
	type candidate struct {
		name string
		role Role
		spec int
		ord  int
	}
	var candidates []candidate
	keys := make([]string, 0, len(c.Components))
	for key := range c.Components {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for ord, name := range keys {
		component := c.Components[name]
		for _, pattern := range component.Paths {
			if glob.Match(pattern, path) {
				candidates = append(candidates, candidate{name: name, role: component.Role, spec: glob.Specificity(pattern), ord: ord})
			}
		}
	}
	if len(candidates) == 0 {
		return Match{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].role == RoleIgnore && candidates[j].role != RoleIgnore {
			return true
		}
		if candidates[j].role == RoleIgnore && candidates[i].role != RoleIgnore {
			return false
		}
		if candidates[i].spec != candidates[j].spec {
			return candidates[i].spec > candidates[j].spec
		}
		return candidates[i].ord < candidates[j].ord
	})
	best := candidates[0]
	return Match{Name: best.name, Role: best.role}, true
}

func (c *Config) IsAllowed(rule, path string) bool {
	for _, allow := range c.Allow {
		if allow.Rule == rule && glob.Match(allow.Path, path) {
			return true
		}
	}
	return false
}

func DiscoverConfig(start string) (string, error) {
	start, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(start)
	if err == nil && !info.IsDir() {
		start = filepath.Dir(start)
	}
	for dir := start; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".hexcheck.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}
	return "", os.ErrNotExist
}
