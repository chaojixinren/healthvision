package tools

import (
	"errors"
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"
)

type reminderTools struct {
	svc                 *services.ReminderService
	requireConfirmation bool
}

type reminderSummary struct {
	ID         uint   `json:"id"`
	UserID     uint   `json:"user_id"`
	MedicineID uint   `json:"medicine_id"`
	Time       string `json:"time"`
	Enabled    bool   `json:"enabled"`
	CreatedBy  uint   `json:"created_by"`
}

func toReminderSummary(r *models.Reminder) reminderSummary {
	if r == nil {
		return reminderSummary{}
	}
	return reminderSummary{
		ID:         r.ID,
		UserID:     r.UserID,
		MedicineID: r.MedicineID,
		Time:       r.Time,
		Enabled:    r.Enabled,
		CreatedBy:  r.CreatedBy,
	}
}

// ---- list ----

type listRemindersArgs struct {
	MedicineID uint `json:"medicine_id,omitempty" jsonschema:"可选，仅列出该药品的提醒"`
	Page       int  `json:"page,omitempty"        jsonschema:"分页页码，默认 1"`
	PerPage    int  `json:"per_page,omitempty"    jsonschema:"每页数量，1-100，默认 20"`
}

type listRemindersResult struct {
	Reminders []reminderSummary `json:"reminders"`
	Total     int64             `json:"total"`
	Page      int               `json:"page"`
	PerPage   int               `json:"per_page"`
}

func (t *reminderTools) list() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:        "list_reminders",
		Description: "列出当前用户的服药提醒。可选按药品 ID 过滤。",
	}, func(ctx tool.Context, in listRemindersArgs) (listRemindersResult, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return listRemindersResult{}, err
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		per := in.PerPage
		if per <= 0 {
			per = 20
		}
		var medFilter *uint
		if in.MedicineID != 0 {
			id := in.MedicineID
			medFilter = &id
		}
		rs, total, err := t.svc.List(ctx, uid, medFilter, page, per)
		if err != nil {
			return listRemindersResult{}, fmt.Errorf("list reminders: %w", err)
		}
		out := listRemindersResult{
			Reminders: make([]reminderSummary, 0, len(rs)),
			Total:     total,
			Page:      page,
			PerPage:   per,
		}
		for i := range rs {
			out.Reminders = append(out.Reminders, toReminderSummary(&rs[i]))
		}
		return out, nil
	})
}

// ---- create ----

type createReminderArgs struct {
	MedicineID   uint   `json:"medicine_id"              jsonschema:"要提醒的药品 ID（必填，必须属于当前用户的药品库）"`
	Time         string `json:"time"                     jsonschema:"提醒时间，24 小时制 HH:MM，例如 08:00 或 20:30"`
	TargetUserID uint   `json:"target_user_id,omitempty" jsonschema:"可选，给绑定的老人创建提醒时填其用户 ID；省略则给当前用户自己创建"`
}

func (t *reminderTools) create() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "create_reminder",
		Description:         "为某药品新建一条服药提醒。子女账户可以传 target_user_id 给已绑定的老人创建。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in createReminderArgs) (reminderSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return reminderSummary{}, err
		}
		if in.MedicineID == 0 {
			return reminderSummary{}, errors.New("medicine_id is required")
		}
		if in.Time == "" {
			return reminderSummary{}, errors.New("time is required (HH:MM)")
		}
		target := uid
		if in.TargetUserID != 0 {
			target = in.TargetUserID
		}
		r, err := t.svc.Create(ctx, uid, target, in.MedicineID, in.Time)
		if err != nil {
			return reminderSummary{}, fmt.Errorf("create reminder: %w", err)
		}
		return toReminderSummary(r), nil
	})
}

// ---- update ----

// updateReminderArgs treats time and enabled as optional. If neither is
// provided we fall back to the current values, which makes the tool tolerant
// to partial inputs from the LLM.
type updateReminderArgs struct {
	ID      uint   `json:"id"                jsonschema:"提醒 ID（必填）"`
	Time    string `json:"time,omitempty"    jsonschema:"新的时间 HH:MM；留空表示保持不变"`
	Enabled *bool  `json:"enabled,omitempty" jsonschema:"是否启用提醒；省略表示保持不变"`
}

func (t *reminderTools) update() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "update_reminder",
		Description:         "修改提醒的时间或启用状态。仅传需要修改的字段。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in updateReminderArgs) (reminderSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return reminderSummary{}, err
		}
		if in.ID == 0 {
			return reminderSummary{}, errors.New("id is required")
		}
		current, err := t.svc.GetByID(ctx, in.ID, uid)
		if err != nil {
			return reminderSummary{}, fmt.Errorf("load reminder: %w", err)
		}
		newTime := pickString(in.Time, current.Time)
		newEnabled := current.Enabled
		if in.Enabled != nil {
			newEnabled = *in.Enabled
		}
		updated, err := t.svc.Update(ctx, in.ID, uid, newTime, newEnabled)
		if err != nil {
			return reminderSummary{}, fmt.Errorf("update reminder: %w", err)
		}
		return toReminderSummary(updated), nil
	})
}

// ---- delete ----

type deleteReminderArgs struct {
	ID uint `json:"id" jsonschema:"要删除的提醒 ID"`
}

func (t *reminderTools) delete_() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "delete_reminder",
		Description:         "删除指定提醒。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in deleteReminderArgs) (deleteResult, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return deleteResult{}, err
		}
		if in.ID == 0 {
			return deleteResult{}, errors.New("id is required")
		}
		if err := t.svc.Delete(ctx, in.ID, uid); err != nil {
			return deleteResult{}, fmt.Errorf("delete reminder: %w", err)
		}
		return deleteResult{Status: "deleted", ID: in.ID}, nil
	})
}
