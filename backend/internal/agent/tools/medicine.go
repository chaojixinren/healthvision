package tools

import (
	"errors"
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"
)

type medicineTools struct {
	svc                 *services.MedicineService
	requireConfirmation bool
}

// medicineSummary is the projection returned to the LLM. We intentionally
// drop verbose fields (timestamps, raw image URL when empty) to keep tool
// outputs token-efficient.
type medicineSummary struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

func toMedicineSummary(m *models.Medicine) medicineSummary {
	if m == nil {
		return medicineSummary{}
	}
	return medicineSummary{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Notes:       m.Notes,
		ImageURL:    m.ImageURL,
	}
}

// ---- list ----

type listMedicinesArgs struct {
	Page    int `json:"page,omitempty"     jsonschema:"分页页码，从 1 开始，默认 1"`
	PerPage int `json:"per_page,omitempty" jsonschema:"每页数量，1-100，默认 20"`
}

type listMedicinesResult struct {
	Medicines []medicineSummary `json:"medicines"`
	Total     int64             `json:"total"`
	Page      int               `json:"page"`
	PerPage   int               `json:"per_page"`
}

func (t *medicineTools) list() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:        "list_medicines",
		Description: "列出当前用户保存的药品。用于回答“我现在有哪些药”、“我有没有阿莫西林”等问题。",
	}, func(ctx tool.Context, in listMedicinesArgs) (listMedicinesResult, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return listMedicinesResult{}, err
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		per := in.PerPage
		if per <= 0 {
			per = 20
		}
		ms, total, err := t.svc.List(ctx, uid, page, per)
		if err != nil {
			return listMedicinesResult{}, fmt.Errorf("list medicines: %w", err)
		}
		out := listMedicinesResult{
			Medicines: make([]medicineSummary, 0, len(ms)),
			Total:     total,
			Page:      page,
			PerPage:   per,
		}
		for i := range ms {
			out.Medicines = append(out.Medicines, toMedicineSummary(&ms[i]))
		}
		return out, nil
	})
}

// ---- get ----

type getMedicineArgs struct {
	ID uint `json:"id" jsonschema:"药品 ID（必填）"`
}

func (t *medicineTools) get() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:        "get_medicine",
		Description: "根据药品 ID 查看详情。修改药品前应先调用此工具确认现有内容。",
	}, func(ctx tool.Context, in getMedicineArgs) (medicineSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return medicineSummary{}, err
		}
		if in.ID == 0 {
			return medicineSummary{}, errors.New("id is required")
		}
		m, err := t.svc.GetByID(ctx, in.ID, uid)
		if err != nil {
			return medicineSummary{}, fmt.Errorf("get medicine: %w", err)
		}
		return toMedicineSummary(m), nil
	})
}

// ---- create ----

type createMedicineArgs struct {
	Name        string `json:"name"                  jsonschema:"药品名称（必填）"`
	Description string `json:"description,omitempty" jsonschema:"药品说明，例如适应症、剂量"`
	Notes       string `json:"notes,omitempty"       jsonschema:"个人备注，例如服用注意事项"`
	ImageURL    string `json:"image_url,omitempty"   jsonschema:"药品图片 URL，可省略"`
}

func (t *medicineTools) create() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "create_medicine",
		Description:         "为当前用户新增药品记录。这是一项写操作，调用前必须确认药名拼写。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in createMedicineArgs) (medicineSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return medicineSummary{}, err
		}
		if in.Name == "" {
			return medicineSummary{}, errors.New("name is required")
		}
		m, err := t.svc.Create(ctx, uid, in.Name, in.ImageURL, in.Description, in.Notes)
		if err != nil {
			return medicineSummary{}, fmt.Errorf("create medicine: %w", err)
		}
		return toMedicineSummary(m), nil
	})
}

// ---- update ----

// updateMedicineArgs uses empty-string semantics to mean "keep existing
// value". This is a deliberate trade-off: the LLM does not need to fetch the
// current record before partial edits, but it cannot clear a field. That
// matches the realistic user intent for medicine entries.
type updateMedicineArgs struct {
	ID          uint   `json:"id"                    jsonschema:"药品 ID（必填）"`
	Name        string `json:"name,omitempty"        jsonschema:"新的药品名称；留空表示保持不变"`
	Description string `json:"description,omitempty" jsonschema:"新的说明；留空表示保持不变"`
	Notes       string `json:"notes,omitempty"       jsonschema:"新的备注；留空表示保持不变"`
	ImageURL    string `json:"image_url,omitempty"   jsonschema:"新的图片 URL；留空表示保持不变"`
}

func (t *medicineTools) update() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "update_medicine",
		Description:         "更新指定药品。仅传需要修改的字段；留空字段保持原值。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in updateMedicineArgs) (medicineSummary, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return medicineSummary{}, err
		}
		if in.ID == 0 {
			return medicineSummary{}, errors.New("id is required")
		}
		current, err := t.svc.GetByID(ctx, in.ID, uid)
		if err != nil {
			return medicineSummary{}, fmt.Errorf("load medicine: %w", err)
		}
		updated, err := t.svc.Update(
			ctx,
			in.ID,
			uid,
			pickString(in.Name, current.Name),
			pickString(in.ImageURL, current.ImageURL),
			pickString(in.Description, current.Description),
			pickString(in.Notes, current.Notes),
		)
		if err != nil {
			return medicineSummary{}, fmt.Errorf("update medicine: %w", err)
		}
		return toMedicineSummary(updated), nil
	})
}

// ---- delete ----

type deleteMedicineArgs struct {
	ID uint `json:"id" jsonschema:"要删除的药品 ID"`
}

type deleteResult struct {
	Status string `json:"status"`
	ID     uint   `json:"id"`
}

func (t *medicineTools) delete_() (tool.Tool, error) {
	return newTool(functiontool.Config{
		Name:                "delete_medicine",
		Description:         "删除指定药品，同时会级联删除该药品的所有提醒。",
		RequireConfirmation: t.requireConfirmation,
	}, func(ctx tool.Context, in deleteMedicineArgs) (deleteResult, error) {
		uid, err := currentUserID(ctx)
		if err != nil {
			return deleteResult{}, err
		}
		if in.ID == 0 {
			return deleteResult{}, errors.New("id is required")
		}
		if err := t.svc.Delete(ctx, in.ID, uid); err != nil {
			return deleteResult{}, fmt.Errorf("delete medicine: %w", err)
		}
		return deleteResult{Status: "deleted", ID: in.ID}, nil
	})
}
