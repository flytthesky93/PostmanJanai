package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/requestassertion"
	"PostmanJanai/ent/requestcapture"
	"PostmanJanai/internal/entity"
	"context"
	"strings"

	"github.com/google/uuid"
)

// RequestRuleRepository persists capture + assertion rules for saved requests (Phase 8).
//
// Both capture and assertion writes use the same "delete-then-create" transactional
// pattern as environment variables, which keeps callers (rule editor in the frontend)
// simple — they always send the full intended list.
type RequestRuleRepository interface {
	ListCaptures(ctx context.Context, requestID string) ([]entity.RequestCaptureRow, error)
	SaveCaptures(ctx context.Context, requestID string, rows []entity.RequestCaptureInput) ([]entity.RequestCaptureRow, error)

	ListAssertions(ctx context.Context, requestID string) ([]entity.RequestAssertionRow, error)
	SaveAssertions(ctx context.Context, requestID string, rows []entity.RequestAssertionInput) ([]entity.RequestAssertionRow, error)
}

type requestRuleRepo struct {
	client *ent.Client
}

func NewRequestRuleRepository(client *ent.Client) RequestRuleRepository {
	return &requestRuleRepo{client: client}
}

func (r *requestRuleRepo) ListCaptures(ctx context.Context, requestID string) ([]entity.RequestCaptureRow, error) {
	uid, err := uuid.Parse(strings.TrimSpace(requestID))
	if err != nil {
		return nil, err
	}
	rows, err := r.client.RequestCapture.Query().
		Where(requestcapture.RequestIDEQ(uid)).
		Order(ent.Asc(requestcapture.FieldSortOrder), ent.Asc(requestcapture.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.RequestCaptureRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.RequestCaptureRow{
			ID:             row.ID.String(),
			Name:           row.Name,
			Source:         row.Source,
			Expression:     row.Expression,
			TargetScope:    row.TargetScope,
			TargetVariable: row.TargetVariable,
			Enabled:        row.Enabled,
			SortOrder:      row.SortOrder,
		})
	}
	return out, nil
}

func (r *requestRuleRepo) SaveCaptures(ctx context.Context, requestID string, rows []entity.RequestCaptureInput) ([]entity.RequestCaptureRow, error) {
	uid, err := uuid.Parse(strings.TrimSpace(requestID))
	if err != nil {
		return nil, err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if _, err := tx.RequestCapture.Delete().Where(requestcapture.RequestIDEQ(uid)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for i, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		target := strings.TrimSpace(row.TargetVariable)
		if target == "" {
			continue
		}
		so := row.SortOrder
		if so == 0 {
			so = i
		}
		_, err := tx.RequestCapture.Create().
			SetRequestID(uid).
			SetName(name).
			SetSource(strings.TrimSpace(row.Source)).
			SetExpression(row.Expression).
			SetTargetScope(strings.TrimSpace(row.TargetScope)).
			SetTargetVariable(target).
			SetEnabled(row.Enabled).
			SetSortOrder(so).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.ListCaptures(ctx, requestID)
}

func (r *requestRuleRepo) ListAssertions(ctx context.Context, requestID string) ([]entity.RequestAssertionRow, error) {
	uid, err := uuid.Parse(strings.TrimSpace(requestID))
	if err != nil {
		return nil, err
	}
	rows, err := r.client.RequestAssertion.Query().
		Where(requestassertion.RequestIDEQ(uid)).
		Order(ent.Asc(requestassertion.FieldSortOrder), ent.Asc(requestassertion.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.RequestAssertionRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.RequestAssertionRow{
			ID:         row.ID.String(),
			Name:       row.Name,
			Source:     row.Source,
			Expression: row.Expression,
			Operator:   row.Operator,
			Expected:   row.Expected,
			Enabled:    row.Enabled,
			SortOrder:  row.SortOrder,
		})
	}
	return out, nil
}

func (r *requestRuleRepo) SaveAssertions(ctx context.Context, requestID string, rows []entity.RequestAssertionInput) ([]entity.RequestAssertionRow, error) {
	uid, err := uuid.Parse(strings.TrimSpace(requestID))
	if err != nil {
		return nil, err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if _, err := tx.RequestAssertion.Delete().Where(requestassertion.RequestIDEQ(uid)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for i, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		so := row.SortOrder
		if so == 0 {
			so = i
		}
		_, err := tx.RequestAssertion.Create().
			SetRequestID(uid).
			SetName(name).
			SetSource(strings.TrimSpace(row.Source)).
			SetExpression(row.Expression).
			SetOperator(strings.TrimSpace(row.Operator)).
			SetExpected(row.Expected).
			SetEnabled(row.Enabled).
			SetSortOrder(so).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.ListAssertions(ctx, requestID)
}
