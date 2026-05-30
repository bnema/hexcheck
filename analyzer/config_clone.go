package analyzer

import "github.com/bnema/hexcheck/config"

func cloneConfig(in *config.Config) *config.Config {
	if in == nil {
		return nil
	}
	out := *in
	out.Components = make(map[string]config.Component, len(in.Components))
	for name, component := range in.Components {
		component.Paths = append([]string(nil), component.Paths...)
		out.Components[name] = component
	}
	out.Rules = make(map[string]config.Severity, len(in.Rules))
	for rule, severity := range in.Rules {
		out.Rules[rule] = severity
	}
	out.ExternalTypes.FrameworkPackages = append([]string(nil), in.ExternalTypes.FrameworkPackages...)
	out.ExternalTypes.AdapterTypePackages = append([]string(nil), in.ExternalTypes.AdapterTypePackages...)
	out.Heuristics.BusinessKeywords = append([]string(nil), in.Heuristics.BusinessKeywords...)
	out.Heuristics.BusinessLogicThreshold = cloneInt(in.Heuristics.BusinessLogicThreshold)
	out.Heuristics.BusinessLogicMinStrongSignals = cloneInt(in.Heuristics.BusinessLogicMinStrongSignals)
	out.Heuristics.BusinessLogicMinWeakSignals = cloneInt(in.Heuristics.BusinessLogicMinWeakSignals)
	out.Heuristics.BusinessLogicMaxFunctionNodes = cloneInt(in.Heuristics.BusinessLogicMaxFunctionNodes)
	out.Heuristics.BusinessLogicMaxDiagnosticsPerPackage = cloneInt(in.Heuristics.BusinessLogicMaxDiagnosticsPerPackage)
	if in.Heuristics.ExcludeTestFiles != nil {
		value := *in.Heuristics.ExcludeTestFiles
		out.Heuristics.ExcludeTestFiles = &value
	}
	out.Mocking.GeneratedMockPaths = append([]string(nil), in.Mocking.GeneratedMockPaths...)
	out.Mocking.GeneratedMockNamePatterns = append([]string(nil), in.Mocking.GeneratedMockNamePatterns...)
	out.RuleSettings = make(map[string]config.RuleSetting, len(in.RuleSettings))
	for rule, setting := range in.RuleSettings {
		setting.ExcludePaths = append([]string(nil), setting.ExcludePaths...)
		out.RuleSettings[rule] = setting
	}
	out.Allow = append([]config.Allow(nil), in.Allow...)
	return &out
}

func cloneInt(in *int) *int {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
