// Package tools provides ADK tool wrappers around HealthVision's business
// services. Each tool is a thin adapter that:
//
//  1. Reads the current user ID from the ADK session (via tool.Context.UserID()).
//  2. Calls the existing service-layer methods so that user isolation,
//     binding checks, time-format validation and other business rules stay
//     in a single place.
//
// Write-side tools opt into Human-in-the-Loop confirmation; the ADK runtime
// will surface a confirmation request to the client before the tool actually
// mutates the database.
package tools

import (
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"healthvision/backend/internal/services"
)

// Deps bundles the service-layer dependencies needed by the tools.
type Deps struct {
	Medicine *services.MedicineService
	Reminder *services.ReminderService

	// RequireWriteConfirmation toggles Human-in-the-Loop confirmation on
	// write-side tools (create/update/delete). Production callers should
	// pass true. Developers can pass false for end-to-end smoke tests so
	// the agent is allowed to mutate data without a confirmation round-trip
	// from the client.
	RequireWriteConfirmation bool
}

// Register builds the full tool set exposed to the agent. It returns an error
// if any tool fails to construct (which would indicate a programming mistake,
// e.g. invalid schema), so callers should treat it as fatal at startup.
func Register(deps Deps) ([]tool.Tool, error) {
	if deps.Medicine == nil || deps.Reminder == nil {
		return nil, errors.New("tools.Register: Medicine and Reminder services are required")
	}

	med := &medicineTools{svc: deps.Medicine, requireConfirmation: deps.RequireWriteConfirmation}
	rem := &reminderTools{svc: deps.Reminder, requireConfirmation: deps.RequireWriteConfirmation}

	builders := []func() (tool.Tool, error){
		med.list,
		med.get,
		med.create,
		med.update,
		med.delete_,
		rem.list,
		rem.create,
		rem.update,
		rem.delete_,
	}

	out := make([]tool.Tool, 0, len(builders))
	for _, b := range builders {
		t, err := b()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// currentUserID extracts the HealthVision user ID from the ADK session.
// The chat service is responsible for setting it via runner.Run(..., userID, ...).
func currentUserID(ctx tool.Context) (uint, error) {
	raw := ctx.UserID()
	if raw == "" {
		return 0, errors.New("会话中缺少用户信息")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("会话中的 user_id %q 无效: %w", raw, err)
	}
	if id == 0 {
		return 0, errors.New("用户 ID 无效")
	}
	return uint(id), nil
}

// pickString returns override if non-empty, otherwise fallback. Used for the
// read-modify-write update pattern: tools accept omitted fields as "keep
// existing value".
func pickString(override, fallback string) string {
	if override == "" {
		return fallback
	}
	return override
}

// newTool is a thin wrapper around functiontool.New that surfaces panics in
// schema inference as descriptive errors at boot time.
func newTool[TArgs, TResults any](cfg functiontool.Config, fn functiontool.Func[TArgs, TResults]) (tool.Tool, error) {
	t, err := functiontool.New(cfg, fn)
	if err != nil {
		return nil, fmt.Errorf("构建工具 %q 失败: %w", cfg.Name, err)
	}
	return t, nil
}
