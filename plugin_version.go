package mtplugins

import (
	"plugin"

	"github.com/Masterminds/semver/v3"
)

type PluginVersion struct {
	Name         string
	Version      string
	Apps         map[string]string // map[AppName]AppVersion
	InitFuncName string

	pluginSymbol *plugin.Plugin
	pluginSemver *semver.Version
}
