package mtplugins

import (
	"log"
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
			log.Fatal("app version is invalid")
			return nil
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
		log.Printf("cannot find init func for plugin: %s %s", pv.Name, pv.Version)
		return initFunc, ErrBadInitFunc
	}

	initFunc, ok := initSymbol.(T)
	if !ok {
		log.Printf("wrong plugin init func declaration for plugin: %s %s", pv.Name, pv.Version)
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
		log.Fatal(err)
		return nil, err
	}
	if len(matches) == 0 {
		log.Fatal(ErrNoPlugins)
		return nil, ErrNoPlugins
	}

	plugins := []*PluginVersion{}
	for _, file := range matches {
		log.Printf("load plugin: %s", file)
		plugin, err := p.loadPlugin(filepath.Join(p.pluginsPath, file))
		if err != nil {
			log.Printf("cannot load plugin %s: %s", file, err)
			continue
		}
		plugins = append(plugins, plugin)
	}

	plugins = p.filterByStages(plugins)

	return p.filterByName(plugins), nil
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
				log.Printf("plugin is not in selected stages: %s %s", plugin.Name, plugin.Version)
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
		log.Printf("error opening so file %s: %s", file, err)
		return nil, ErrNotAPlugin
	}
	versionSymbol, err := plugin.Lookup(PLUGIN_VERSION_FUNC)
	if err != nil {
		log.Printf("cannot find version func in so file: %s", file)
		return nil, ErrCannotGetVer
	}
	versionFunc, ok := versionSymbol.(func() *PluginVersion)
	if !ok {
		log.Printf("wrong plugin version func declaration in so file: %s", file)
		return nil, ErrCannotGetVer
	}
	version := versionFunc()
	version.pluginSymbol = plugin
	version.pluginSemver, err = semver.NewVersion(version.Version)
	if err != nil {
		log.Printf("plugin version is invalid: %s %s", version.Name, version.Version)
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
		log.Printf("cannot parse app version constraints in plugin: %s", appVer)
		return ErrBadPluginVer
	}
	if !ver.Check(p.appVersion) {
		log.Printf("plugin is not compatible with current app version : %s", appVer)
		return ErrBadAppVersion
	}
	return nil
}
