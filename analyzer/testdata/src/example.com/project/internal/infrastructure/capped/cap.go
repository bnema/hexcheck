package capped

import "example.com/project/internal/domain"

func first(disabled bool) error { // want "suspicious-business-logic-in-adapter"
	if disabled {
		return domain.ErrDisabled
	}
	if !disabled {
		return nil
	}
	return nil
}

//hexcheck:ignore suspicious-business-logic-in-adapter intentional policy at the boundary
func ignoredAfterCap(disabled bool) error {
	if disabled {
		return domain.ErrDisabled
	}
	if !disabled {
		return nil
	}
	return nil
}

func hiddenOne(disabled bool) error { // want "business-logic-diagnostics-hidden: 2 more suspicious-business-logic-in-adapter diagnostics hidden by heuristics.businessLogicMaxDiagnosticsPerPackage"
	if disabled {
		return domain.ErrDisabled
	}
	if !disabled {
		return nil
	}
	return nil
}
func hiddenTwo(disabled bool) error {
	if disabled {
		return domain.ErrDisabled
	}
	if !disabled {
		return nil
	}
	return nil
}

//hexcheck:ignore suspicious-business-logic-in-adapter obsolete exemption // want "unused-ignore-directive"
func clean() {}
