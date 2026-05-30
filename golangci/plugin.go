package golangci

import (
	"fmt"

	"github.com/bnema/hexcheck/analyzer"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin(analyzer.Name, New)
}

func New(settings any) (register.LinterPlugin, error) {
	cfg, err := parseSettings(settings)
	if err != nil {
		return nil, err
	}
	return plugin{configPath: cfg.ConfigPath, root: cfg.Root, modulePath: cfg.ModulePath}, nil
}

type plugin struct {
	configPath string
	root       string
	modulePath string
}

func (p plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{analyzer.New(analyzer.Options{
		ConfigPath: p.configPath,
		Root:       p.root,
		ModulePath: p.modulePath,
	})}, nil
}

func (p plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

type settingsConfig struct {
	ConfigPath string
	Root       string
	ModulePath string
}

func parseSettings(settings any) (settingsConfig, error) {
	if settings == nil {
		return settingsConfig{}, nil
	}
	m, ok := settings.(map[string]any)
	if !ok {
		return settingsConfig{}, fmt.Errorf("hexcheck settings must be a map, got %T", settings)
	}
	return settingsConfig{
		ConfigPath: stringSetting(m, "config"),
		Root:       stringSetting(m, "root"),
		ModulePath: stringSetting(m, "module"),
	}, nil
}

func stringSetting(settings map[string]any, key string) string {
	value, ok := settings[key]
	if !ok || value == nil {
		return ""
	}
	s, _ := value.(string)
	return s
}
