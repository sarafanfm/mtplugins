package mtplugins

import "errors"

const (
	ERR_NO_PLUGINS      = "no plugins found"
	ERR_NOT_A_PLUGIN    = "not a plugin"
	ERR_CANNOT_GET_VER  = "cannot get plugin version"
	ERR_NOT_AN_APP      = "plugin not for this app"
	ERR_BAD_PLUGIN_VER  = "plugin version is invalid"
	ERR_BAD_APP_VERSION = "incompatible app version for plugin"
	ERR_BAD_INIT_FUNC   = "cannot find init func for plugin"
	ERR_BAD_INIT_TYPE   = "wrong plugin init func declaration"
)

var (
	ErrNoPlugins     = errors.New(ERR_NO_PLUGINS)
	ErrNotAPlugin    = errors.New(ERR_NOT_A_PLUGIN)
	ErrCannotGetVer  = errors.New(ERR_CANNOT_GET_VER)
	ErrNotAnApp      = errors.New(ERR_NOT_AN_APP)
	ErrBadPluginVer  = errors.New(ERR_BAD_PLUGIN_VER)
	ErrBadAppVersion = errors.New(ERR_BAD_APP_VERSION)
	ErrBadInitFunc   = errors.New(ERR_BAD_INIT_FUNC)
	ErrBadInitType   = errors.New(ERR_BAD_INIT_TYPE)
)
