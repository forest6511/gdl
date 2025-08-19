package hooks

import (
	"context"

	"github.com/forest6511/gdl/pkg/types"
)

// HookType represents different types of hooks in the system
type HookType = types.HookType

const (
	HookPreDownload  = types.PreDownloadHook
	HookPostDownload = types.PostDownloadHook
	HookPreChunk     = types.CustomHook
	HookPostChunk    = types.CustomHook
	HookOnError      = types.ErrorHook
	HookOnRetry      = types.CustomHook
	HookOnProgress   = types.CustomHook
	HookOnComplete   = types.PostDownloadHook
)

// HookContext provides context and data for hook execution
type HookContext struct {
	Type     HookType
	Data     interface{}
	Metadata map[string]interface{}
	Cancel   context.CancelFunc
}

// HookHandler represents a function that handles a specific hook
type HookHandler func(ctx context.Context, hookCtx *HookContext) error

// HookExecutor interface for executing and registering hooks
type HookExecutor interface {
	Execute(ctx context.Context, hook *HookContext) error
	Register(hookType HookType, handler HookHandler) error
}
