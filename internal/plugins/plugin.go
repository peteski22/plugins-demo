package plugins

import (
	pkg "github.com/peteski22/plugins-demo/pkg/contract/plugin"
)

// PluginInstance represents an instance of a Plugin to the Manager.
// This encapsulates the plugin, any configuration it should be supplied,
// and whether the plugin is required to succeed.
// NOTE: Use NewPluginInstance to create a PluginInstance
type PluginInstance struct {
	pkg.Plugin

	config   pkg.PluginConfig
	id       string
	required bool // TODO: this should be something that the pipeline cares about based on config...
}

// TODO: Needs 'id' param, but TODO: too many params for func... all required, urghh
//// NewPluginInstance creates a new PluginInstance.
//func NewPluginInstance(p pkg.Plugin, cfg pkg.Config, required bool) *PluginInstance {
//	return &PluginInstance{
//		Plugin:   p,
//		config:   cfg,
//		required: required,
//	}
//}

func (pi *PluginInstance) ID() string {
	return pi.id
}

//func (pi *PluginInstance) Name() string {
//	return pi.Metadata().Name
//}

func (pi *PluginInstance) Required() bool { return pi.required }

func (pi *PluginInstance) CanHandle(f pkg.Flow) bool {
	_, ok := pi.Capabilities()[f]
	return ok
}
