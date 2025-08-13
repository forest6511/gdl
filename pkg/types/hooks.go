package types

// HookType represents different types of plugin hooks with unified string-based approach
type HookType string

const (
	// Download lifecycle hooks
	PreDownloadHook  HookType = "pre_download"
	PostDownloadHook HookType = "post_download"

	// Storage hooks
	PreStoreHook  HookType = "pre_store"
	PostStoreHook HookType = "post_store"

	// Authentication hooks
	AuthHook HookType = "auth"

	// Data transformation hooks
	TransformHook HookType = "transform"

	// Plugin lifecycle hooks
	PluginInitHook    HookType = "plugin_init"
	PluginCleanupHook HookType = "plugin_cleanup"

	// Error handling hooks
	ErrorHook HookType = "error"

	// Custom hooks (allow plugins to define custom hooks)
	CustomHook HookType = "custom"
)

// HookPriority defines the execution priority of hooks
type HookPriority int

const (
	PriorityHigh   HookPriority = 100
	PriorityNormal HookPriority = 50
	PriorityLow    HookPriority = 10
)

// HookContext provides context information for hook execution
type HookContext struct {
	HookType HookType               `json:"hook_type"`
	Data     map[string]interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Priority HookPriority           `json:"priority"`
}

// HookExecutor defines the interface for executing hooks
type HookExecutor interface {
	Execute(ctx *HookContext) error
	GetPriority() HookPriority
	GetName() string
}
