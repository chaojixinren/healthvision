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
	ID           uint   `json:"id"`
	UserID       uint   `json:"user_id"`
	MedicineID   uint   `json:"medicine_id"`
	Time         string `json:"time"`
	RepeatType   string `json:"repeat_type"`
	IntervalDays int    `json:"interval_days"`
	Weekdays     string `json:"weekdays"`
	Enabled      bool   `json:"enabled"`
	CreatedBy    uint   `json:"created_by"`
}

func toReminderSummary(r *models.Reminder) reminderSummary {
	if r == nil {
		return reminderSummary{}
	}
	return reminderSummary{
		ID:           r.ID,
		UserID:       r.UserID,
		MedicineID:   r.MedicineID,
		Time:         r.Time,
		RepeatType:   r.RepeatType,
		IntervalDays: r.IntervalDays,
		Weekdays:     r.Weekdays,
		Enabled:      r.Enabled,
		CreatedBy:    r.CreatedBy,
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
			return listRemindersResult{}, fmt.Errorf("获取提醒列表失败: %w", err)
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
	RepeatType   string `json:"repeat_type,omitempty"    jsonschema:"重复类型：daily(每天)/interval(每隔N天)/weekly(每周固定)，默认 daily"`
	IntervalDays int    `json:"interval_days,omitempty"  jsonschema:"间隔天数，仅 repeat_type=interval 时有效，最小 2（隔天）"`
	Weekdays     string `json:"weekdays,omitempty"       jsonschema:"星期几，仅 repeat_type=weekly 时有效，逗号分隔的 0-6（0=周日 1=周一 … 6=周六），例如 1,3,5 表示周一三五"`
}

func (t *reminderTools) create() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "create_reminder",
		Description:         "为某药品新建一条服药提醒。子女账户可以传 target_user_id 给已绑定的老人创建。支持 daily（每天）、interval（每隔N天，如 interval_days=2 为隔天）、weekly（每周固定星期，如 weekdays=1,3,5 为周一三五）。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in createReminderArgs) (reminderSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return reminderSummary{}, err
		}
		if in.MedicineID == 0 {
			return reminderSummary{}, errors.New("缺少药品 ID")
		}
		if in.Time == "" {
			return reminderSummary{}, errors.New("缺少提醒时间（格式：HH:MM）")
		}
		target := uid
		if in.TargetUserID != 0 {
			target = in.TargetUserID
		}
		r, err := t.svc.Create(ctx, uid, target, in.MedicineID, in.Time, in.RepeatType, in.IntervalDays, in.Weekdays)
		if err != nil {
			return reminderSummary{}, fmt.Errorf("创建提醒失败: %w", err)
		}
		return toReminderSummary(r), nil
	})
}

// ---- update ----

type updateReminderArgs struct {
	ID           uint   `json:"id"                       jsonschema:"提醒 ID（必填）"`
	Time         string `json:"time,omitempty"           jsonschema:"新的时间 HH:MM；留空表示保持不变"`
	Enabled      *bool  `json:"enabled,omitempty"        jsonschema:"是否启用提醒；省略表示保持不变"`
	RepeatType   string `json:"repeat_type,omitempty"    jsonschema:"重复类型：daily/interval/weekly；留空表示保持不变"`
	IntervalDays int    `json:"interval_days,omitempty"  jsonschema:"间隔天数，仅 repeat_type=interval 时有效"`
	Weekdays     string `json:"weekdays,omitempty"       jsonschema:"星期几，仅 repeat_type=weekly 时有效"`
}

func (t *reminderTools) update() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "update_reminder",
		Description:         "修改提醒的时间、重复规则或启用状态。仅传需要修改的字段。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in updateReminderArgs) (reminderSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return reminderSummary{}, err
		}
		if in.ID == 0 {
			return reminderSummary{}, errors.New("缺少提醒 ID")
		}
		current, err := t.svc.GetByID(ctx, in.ID, uid)
		if err != nil {
			return reminderSummary{}, fmt.Errorf("加载提醒失败: %w", err)
		}
		newTime := pickString(in.Time, current.Time)
		newEnabled := current.Enabled
		if in.Enabled != nil {
			newEnabled = *in.Enabled
		}
		newRepeatType := pickString(in.RepeatType, current.RepeatType)
		newIntervalDays := current.IntervalDays
		if in.RepeatType == models.RepeatTypeInterval && in.IntervalDays > 0 {
			newIntervalDays = in.IntervalDays
		} else if in.RepeatType != "" && in.RepeatType != models.RepeatTypeInterval {
			newIntervalDays = 0
		}
		newWeekdays := current.Weekdays
		if in.RepeatType == models.RepeatTypeWeekly && in.Weekdays != "" {
			newWeekdays = in.Weekdays
		} else if in.RepeatType != "" && in.RepeatType != models.RepeatTypeWeekly {
			newWeekdays = ""
		}
		updated, err := t.svc.Update(ctx, in.ID, uid, newTime, newEnabled, newRepeatType, newIntervalDays, newWeekdays)
		if err != nil {
			return reminderSummary{}, fmt.Errorf("更新提醒失败: %w", err)
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
			return deleteResult{}, errors.New("缺少提醒 ID")
		}
		if err := t.svc.Delete(ctx, in.ID, uid); err != nil {
			return deleteResult{}, fmt.Errorf("删除提醒失败: %w", err)
		}
		return deleteResult{Status: "deleted", ID: in.ID}, nil
	})
}
