package utils

import "runtime/debug"

const modulePath = "github.com/coreofscience/go-bibx"

// ModuleVersion reports the version of the go-bibx module in the running
// binary, whether it is the main module or a dependency.
func ModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	if info.Main.Path == modulePath {
		return info.Main.Version
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			return dep.Version
		}
	}
	return "(devel)"
}
