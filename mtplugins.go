package mtplugins

import (
	"path/filepath"
	"plugin"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/sarafanfm/mtserver"
)

type ReleaseStage string

const (
	RELEASE_STAGE_DEV    ReleaseStage = "dev"
	RELEASE_STAGE_ALPHA  ReleaseStage = "alpha"
	RELEASE_STAGE_BETA   ReleaseStage = "beta"
	RELEASE_STAGE_RC     ReleaseStage = "rc"
	RELEASE_STAGE_STABLE ReleaseStage = ""
)

const PLUGIN_PATTERN = "*.so"
const PLUGIN_VERSION_FUNC = "GetVersion"

const (
	DEFAULT_PLUGINS_PATH = "./plugins"
)

type MTPlugins struct {
	appName       string
	appVersion    *semver.Version
	pluginsStages []ReleaseStage

	pluginsPath string

	mtServer *mtserver.MTServer
}

func New(appName, appVersion, pluginsPath string, pluginsStages []ReleaseStage) *MTPlugins {
	appName = strings.TrimSpace(appName)

	appVersion = strings.TrimSpace(appVersion)
	var appVer *semver.Version = nil
	if appVersion != "" {
		var err error
		appVer, err = semver.NewVersion(appVersion)
		if err != nil {
			panic(err)
		}
	}

	if pluginsPath == "" {
		pluginsPath = DEFAULT_PLUGINS_PATH
	}

	if pluginsStages == nil {
		pluginsStages = []ReleaseStage{RELEASE_STAGE_STABLE}
	}

	return &MTPlugins{
		appName:       appName,
		appVersion:    appVer,
		pluginsPath:   pluginsPath,
		pluginsStages: pluginsStages,

		mtServer: mtserver.New(),
	}
}

func GetInitFunc[T any](pv *PluginVersion) (T, error) {
	var initFunc T

	initSymbol, err := pv.pluginSymbol.Lookup(pv.InitFuncName)
	if err != nil {
		return initFunc, ErrBadInitFunc
	}

	initFunc, ok := initSymbol.(T)
	if !ok {
		return initFunc, ErrBadInitType
	}

	return initFunc, nil
}

func (p *MTPlugins) AddEndpoint(name string, opts *mtserver.EndpointOpts) *mtserver.Endpoint {
	return p.mtServer.AddEndpoint(name, opts)
}

func (p *MTPlugins) Run() {
	p.mtServer.Run()
}

func (p *MTPlugins) Load() ([]*PluginVersion, error) {
	matches, err := filepath.Glob(filepath.Join(p.pluginsPath, PLUGIN_PATTERN))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, ErrNoPlugins
	}

	plugins := []*PluginVersion{}
	for _, file := range matches {
		plugin, err := p.loadPlugin(file)
		if err != nil {
			continue
		}
		plugins = append(plugins, plugin)
	}

	plugins = p.filterByName(p.filterByStages(plugins))
	
	if len(plugins) == 0 {
		return nil, ErrNoPlugins
	}

	return plugins, nil
}

func (p *MTPlugins) filterByStages(plugins []*PluginVersion) []*PluginVersion {
	filtered := []*PluginVersion{}
	for _, plugin := range plugins {
		if len(p.pluginsStages) > 0 {
			var found bool = false
			for _, stage := range p.pluginsStages {
				if len(string(stage)) == 0 { // RELEASE_STAGE_STABLE
					if plugin.pluginSemver.Prerelease() == "" {
						found = true
						break
					} else {
						continue
					}
				}

				if strings.HasPrefix(plugin.pluginSemver.Prerelease(), string(stage)) { // TODO: strings.Contains ??
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, plugin)
	}
	return filtered
}

func (p *MTPlugins) filterByName(plugins []*PluginVersion) []*PluginVersion {
	filtered := []*PluginVersion{}
	known := map[string]*PluginVersion{}
	for _, plugin := range plugins {
		if exists, ok := known[plugin.Name]; !ok {
			known[plugin.Name] = plugin
		} else {
			if plugin.pluginSemver.GreaterThan(exists.pluginSemver) {
				known[plugin.Name] = plugin
			}
		}
	}

	for _, plugin := range known {
		filtered = append(filtered, plugin)
	}

	return filtered
}

func (p *MTPlugins) loadPlugin(file string) (*PluginVersion, error) {
	plugin, err := plugin.Open(file)
	if err != nil {
		return nil, ErrNotAPlugin
	}
	versionSymbol, err := plugin.Lookup(PLUGIN_VERSION_FUNC)
	if err != nil {
		return nil, ErrCannotGetVer
	}
	versionFunc, ok := versionSymbol.(func() *PluginVersion)
	if !ok {
		return nil, ErrCannotGetVer
	}
	version := versionFunc()
	version.pluginSymbol = plugin
	version.pluginSemver, err = semver.NewVersion(version.Version)
	if err != nil {
		return nil, ErrBadPluginVer
	}

	if err := p.checkIfAppSatisfied(version); err != nil {
		return nil, err
	}

	return version, nil
}

func (p *MTPlugins) checkIfAppSatisfied(plugin *PluginVersion) error {
	appVer, ok := plugin.Apps[p.appName]
	if !ok {
		if p.appName == "" && p.appVersion == nil {
			return nil
		}
		return ErrNotAnApp
	}

	ver, err := semver.NewConstraint(appVer)
	if err != nil {
		return ErrBadPluginVer
	}
	if !ver.Check(p.appVersion) {
		return ErrBadAppVersion
	}
	return nil
}
