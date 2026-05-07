package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"

	"healthvision/backend/internal/models"
)

// MedicineLister is the minimal read-only interface the instruction provider
// needs from the medicine service. We declare it locally instead of importing
// the services package because services already imports this package
// (chat_service references agent.Name), so a direct dependency would create
// an import cycle. *services.MedicineService satisfies this interface
// implicitly.
type MedicineLister interface {
	List(ctx context.Context, userID uint, page, perPage int) ([]models.Medicine, int64, error)
}

// ReminderLister mirrors MedicineLister for reminders.
type ReminderLister interface {
	List(ctx context.Context, userID uint, medicineID *uint, page, perPage int) ([]models.Reminder, int64, error)
}

// maxSnapshotItems caps the number of medicine and reminder rows we inject
// into the system prompt. This bounds the token cost per turn and stays well
// under realistic per-user volumes; if a user ever exceeds it the model can
// still call list_* tools to page through.
const maxSnapshotItems = 30

// NewDynamicInstruction returns an InstructionProvider that combines the
// static behavioural prompt with a freshly-read snapshot of the caller's
// medicine library and reminder schedule. ADK invokes the provider once per
// agent invocation, before any LLM call, so the model receives current state
// without an extra list_* round-trip on common questions.
//
// On any failure (missing user_id in session, DB error, etc.) the provider
// degrades gracefully back to the static prompt so a transient outage in
// these auxiliary reads cannot break the chat flow.
func NewDynamicInstruction(med MedicineLister, rem ReminderLister) llmagent.InstructionProvider {
	return func(rc agent.ReadonlyContext) (string, error) {
		uid, ok := userIDFromContext(rc)
		if !ok {
			return defaultInstruction, nil
		}
		var b strings.Builder
		b.WriteString(defaultInstruction)
		b.WriteString("\n\n----\n\n以下是当前用户的最新数据快照（每次对话开始时自动注入）。\n")
		b.WriteString("简单查询请直接基于此回答，无需再调用 list_medicines / list_reminders；\n")
		b.WriteString("修改或删除前请以本快照中的 ID 为准。\n")
		appendMedicineSection(&b, rc, med, uid)
		appendReminderSection(&b, rc, rem, uid)
		return b.String(), nil
	}
}

func userIDFromContext(rc agent.ReadonlyContext) (uint, bool) {
	raw := rc.UserID()
	if raw == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

func appendMedicineSection(b *strings.Builder, ctx context.Context, svc MedicineLister, uid uint) {
	b.WriteString("\n【当前用户的药品库】\n")
	meds, _, err := svc.List(ctx, uid, 1, maxSnapshotItems)
	if err != nil {
		b.WriteString("（读取药品库失败，请通过 list_medicines 工具查询）\n")
		return
	}
	if len(meds) == 0 {
		b.WriteString("（暂无药品）\n")
		return
	}
	for i := range meds {
		m := &meds[i]
		line := fmt.Sprintf("- [ID=%d] %s", m.ID, m.Name)
		if d := strings.TrimSpace(m.Description); d != "" {
			line += " — " + truncateRunes(d, 40)
		}
		if n := strings.TrimSpace(m.Notes); n != "" {
			line += "（备注：" + truncateRunes(n, 30) + "）"
		}
		b.WriteString(line + "\n")
	}
}

func appendReminderSection(b *strings.Builder, ctx context.Context, svc ReminderLister, uid uint) {
	b.WriteString("\n【当前用户的服药提醒】\n")
	rems, _, err := svc.List(ctx, uid, nil, 1, maxSnapshotItems)
	if err != nil {
		b.WriteString("（读取提醒失败，请通过 list_reminders 工具查询）\n")
		return
	}
	if len(rems) == 0 {
		b.WriteString("（暂无提醒）\n")
		return
	}
	for i := range rems {
		r := &rems[i]
		status := "已启用"
		if !r.Enabled {
			status = "已关闭"
		}
		repeat := formatRepeatLabel(r.RepeatType, r.IntervalDays, r.Weekdays)
		b.WriteString(fmt.Sprintf("- [提醒 ID=%d] 药品 ID=%d 时间 %s %s（%s）\n",
			r.ID, r.MedicineID, r.Time, repeat, status))
	}
}

func formatRepeatLabel(repeatType string, intervalDays int, weekdays string) string {
	switch repeatType {
	case models.RepeatTypeInterval:
		return fmt.Sprintf("（每%d天）", intervalDays)
	case models.RepeatTypeWeekly:
		if weekdays == "" {
			return ""
		}
		labels := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
		var parts []string
		for _, s := range strings.Split(weekdays, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err == nil && n >= 0 && n <= 6 {
				parts = append(parts, labels[n])
			}
		}
		return "（" + strings.Join(parts, "、") + "）"
	default:
		return "（每天）"
	}
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
