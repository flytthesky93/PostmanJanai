package repository

import (
	"PostmanJanai/ent"
	"PostmanJanai/ent/request"
	"PostmanJanai/ent/requestassertion"
	"PostmanJanai/ent/requestcapture"
	"PostmanJanai/ent/requestformfield"
	"PostmanJanai/ent/requestheader"
	"PostmanJanai/ent/requestqueryparam"
	"PostmanJanai/internal/entity"
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"
)

const (
	fieldKindURLEncoded    = "urlencoded"
	fieldKindMultipartText = "multipart_text"
	fieldKindMultipartFile = "multipart_file"
)

// RequestRepository persists saved HTTP requests and child rows (headers, query, form/multipart).
type RequestRepository interface {
	CreateFull(ctx context.Context, in *entity.SavedRequestFull) (string, error)
	UpdateFull(ctx context.Context, in *entity.SavedRequestFull) error
	DeleteByID(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*entity.SavedRequestFull, error)
	ListByFolder(ctx context.Context, folderID string) ([]*entity.SavedRequestSummary, error)
	ExistsNameInFolder(ctx context.Context, folderID, name string, excludeRequestID *string) (bool, error)

	// SearchByNameOrURL returns saved requests whose name OR URL matches the
	// `query` (case-insensitive substring). Capped at `limit` rows; the second
	// return value signals truncation.
	SearchByNameOrURL(ctx context.Context, query string, limit int) ([]*entity.SavedRequestSummary, bool, error)

	// MoveToFolder updates `folder_id` for a saved request. Caller validates
	// name uniqueness within the destination folder.
	MoveToFolder(ctx context.Context, requestID, folderID string) error
}

type requestRepo struct {
	client *ent.Client
}

func NewRequestRepository(client *ent.Client) RequestRepository {
	return &requestRepo{client: client}
}

func (r *requestRepo) ExistsNameInFolder(ctx context.Context, folderID, name string, excludeRequestID *string) (bool, error) {
	fid, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return false, err
	}
	name = strings.TrimSpace(name)
	q := r.client.Request.Query().
		Where(
			request.FolderIDEQ(fid),
			request.NameEQ(name),
		)
	if excludeRequestID != nil && strings.TrimSpace(*excludeRequestID) != "" {
		ex, err := uuid.Parse(strings.TrimSpace(*excludeRequestID))
		if err != nil {
			return false, err
		}
		q = q.Where(request.IDNEQ(ex))
	}
	return q.Exist(ctx)
}

func (r *requestRepo) SearchByNameOrURL(ctx context.Context, query string, limit int) ([]*entity.SavedRequestSummary, bool, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, false, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.client.Request.Query().
		Where(request.Or(
			request.NameContainsFold(q),
			request.URLContainsFold(q),
		)).
		Order(ent.Desc(request.FieldUpdatedAt)).
		Limit(limit + 1).
		All(ctx)
	if err != nil {
		return nil, false, err
	}
	truncated := len(rows) > limit
	if truncated {
		rows = rows[:limit]
	}
	return mapSummaries(rows), truncated, nil
}

func (r *requestRepo) MoveToFolder(ctx context.Context, requestID, folderID string) error {
	rid, err := uuid.Parse(strings.TrimSpace(requestID))
	if err != nil {
		return err
	}
	fid, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return err
	}
	return r.client.Request.UpdateOneID(rid).SetFolderID(fid).Exec(ctx)
}

func (r *requestRepo) ListByFolder(ctx context.Context, folderID string) ([]*entity.SavedRequestSummary, error) {
	fid, err := uuid.Parse(strings.TrimSpace(folderID))
	if err != nil {
		return nil, err
	}
	rows, err := r.client.Request.Query().
		Where(request.FolderIDEQ(fid)).
		Order(ent.Desc(request.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return mapSummaries(rows), nil
}

func mapSummaries(rows []*ent.Request) []*entity.SavedRequestSummary {
	out := make([]*entity.SavedRequestSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.SavedRequestSummary{
			ID:        row.ID.String(),
			FolderID:  row.FolderID.String(),
			Name:      row.Name,
			Method:    row.Method,
			URL:       row.URL,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out
}

func (r *requestRepo) DeleteByID(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	if err := deleteRequestParts(ctx, tx, uid); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := deleteRequestRules(ctx, tx, uid); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Request.DeleteOneID(uid).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func deleteRequestParts(ctx context.Context, tx *ent.Tx, requestID uuid.UUID) error {
	if _, err := tx.RequestHeader.Delete().Where(requestheader.RequestIDEQ(requestID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.RequestQueryParam.Delete().Where(requestqueryparam.RequestIDEQ(requestID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.RequestFormField.Delete().Where(requestformfield.RequestIDEQ(requestID)).Exec(ctx); err != nil {
		return err
	}
	return nil
}

// deleteRequestRules removes Phase 8 capture + assertion rules. These are managed
// independently of the request payload (own SaveCaptures / SaveAssertions APIs),
// so callers should ONLY invoke this from a full request deletion path — never
// from UpdateFull, which would silently wipe the user's rules.
func deleteRequestRules(ctx context.Context, tx *ent.Tx, requestID uuid.UUID) error {
	if _, err := tx.RequestCapture.Delete().Where(requestcapture.RequestIDEQ(requestID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.RequestAssertion.Delete().Where(requestassertion.RequestIDEQ(requestID)).Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (r *requestRepo) GetByID(ctx context.Context, id string) (*entity.SavedRequestFull, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.client.Request.Query().
		Where(request.IDEQ(uid)).
		WithRequestHeaders().
		WithRequestQueryParams().
		WithRequestFormFields().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return entRequestToFull(row), nil
}

func entRequestToFull(row *ent.Request) *entity.SavedRequestFull {
	out := &entity.SavedRequestFull{
		ID:        row.ID.String(),
		FolderID:  row.FolderID.String(),
		Name:      row.Name,
		Method:    row.Method,
		URL:       row.URL,
		BodyMode:  row.BodyMode,
		InsecureSkipVerify: row.InsecureSkipVerify,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.RawBody != nil {
		t := *row.RawBody
		out.RawBody = &t
	}
	if row.AuthJSON != nil && strings.TrimSpace(*row.AuthJSON) != "" {
		var a entity.RequestAuth
		if err := json.Unmarshal([]byte(*row.AuthJSON), &a); err == nil {
			out.Auth = &a
		}
	}

	hh := row.Edges.RequestHeaders
	sort.Slice(hh, func(i, j int) bool { return hh[i].SortOrder < hh[j].SortOrder })
	for _, h := range hh {
		out.Headers = append(out.Headers, entity.KeyValue{Key: h.Key, Value: h.Value})
	}

	qq := row.Edges.RequestQueryParams
	sort.Slice(qq, func(i, j int) bool { return qq[i].SortOrder < qq[j].SortOrder })
	for _, q := range qq {
		out.QueryParams = append(out.QueryParams, entity.KeyValue{Key: q.Key, Value: q.Value})
	}

	ff := row.Edges.RequestFormFields
	sort.Slice(ff, func(i, j int) bool { return ff[i].SortOrder < ff[j].SortOrder })
	for _, f := range ff {
		switch f.FieldKind {
		case fieldKindURLEncoded:
			v := ""
			if f.Value != nil {
				v = *f.Value
			}
			out.FormFields = append(out.FormFields, entity.KeyValue{Key: f.Key, Value: v})
		case fieldKindMultipartText:
			v := ""
			if f.Value != nil {
				v = *f.Value
			}
			out.MultipartParts = append(out.MultipartParts, entity.MultipartPart{Key: f.Key, Kind: "text", Value: v})
		case fieldKindMultipartFile:
			if f.Value != nil {
				out.MultipartParts = append(out.MultipartParts, entity.MultipartPart{Key: f.Key, Kind: "file", FilePath: *f.Value})
			}
		default:
			v := ""
			if f.Value != nil {
				v = *f.Value
			}
			out.FormFields = append(out.FormFields, entity.KeyValue{Key: f.Key, Value: v})
		}
	}
	return out
}

func (r *requestRepo) CreateFull(ctx context.Context, in *entity.SavedRequestFull) (string, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return "", err
	}
	fid, err := uuid.Parse(strings.TrimSpace(in.FolderID))
	if err != nil {
		_ = tx.Rollback()
		return "", err
	}
	b := tx.Request.Create().
		SetFolderID(fid).
		SetName(strings.TrimSpace(in.Name)).
		SetMethod(strings.TrimSpace(in.Method)).
		SetURL(strings.TrimSpace(in.URL)).
		SetBodyMode(strings.TrimSpace(in.BodyMode)).
		SetInsecureSkipVerify(in.InsecureSkipVerify)
	if strings.TrimSpace(in.BodyMode) == "" {
		b = b.SetBodyMode("none")
	}
	if in.RawBody != nil {
		b = b.SetRawBody(*in.RawBody)
	}
	if in.Auth != nil {
		if raw, err := json.Marshal(in.Auth); err == nil {
			s := string(raw)
			b = b.SetAuthJSON(s)
		}
	}
	row, err := b.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return "", err
	}
	if err := createRequestParts(ctx, tx, row.ID, in); err != nil {
		_ = tx.Rollback()
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return row.ID.String(), nil
}

func (r *requestRepo) UpdateFull(ctx context.Context, in *entity.SavedRequestFull) error {
	uid, err := uuid.Parse(strings.TrimSpace(in.ID))
	if err != nil {
		return err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	if err := deleteRequestParts(ctx, tx, uid); err != nil {
		_ = tx.Rollback()
		return err
	}
	fid, err := uuid.Parse(strings.TrimSpace(in.FolderID))
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	up := tx.Request.UpdateOneID(uid).
		SetFolderID(fid).
		SetName(strings.TrimSpace(in.Name)).
		SetMethod(strings.TrimSpace(in.Method)).
		SetURL(strings.TrimSpace(in.URL)).
		SetBodyMode(strings.TrimSpace(in.BodyMode)).
		SetInsecureSkipVerify(in.InsecureSkipVerify)
	if strings.TrimSpace(in.BodyMode) == "" {
		up = up.SetBodyMode("none")
	}
	if in.RawBody != nil {
		up = up.SetRawBody(*in.RawBody)
	} else {
		up = up.ClearRawBody()
	}
	if in.Auth != nil {
		if raw, err := json.Marshal(in.Auth); err == nil {
			s := string(raw)
			up = up.SetAuthJSON(s)
		}
	} else {
		up = up.ClearAuthJSON()
	}
	if err := up.Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := createRequestParts(ctx, tx, uid, in); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func createRequestParts(ctx context.Context, tx *ent.Tx, requestID uuid.UUID, in *entity.SavedRequestFull) error {
	for i, h := range in.Headers {
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		_, err := tx.RequestHeader.Create().
			SetRequestID(requestID).
			SetKey(k).
			SetValue(h.Value).
			SetEnabled(true).
			SetSortOrder(i).
			Save(ctx)
		if err != nil {
			return err
		}
	}
	for i, q := range in.QueryParams {
		k := strings.TrimSpace(q.Key)
		if k == "" {
			continue
		}
		_, err := tx.RequestQueryParam.Create().
			SetRequestID(requestID).
			SetKey(k).
			SetValue(q.Value).
			SetEnabled(true).
			SetSortOrder(i).
			Save(ctx)
		if err != nil {
			return err
		}
	}
	sortOrder := 0
	for _, kv := range in.FormFields {
		k := strings.TrimSpace(kv.Key)
		if k == "" {
			continue
		}
		v := kv.Value
		_, err := tx.RequestFormField.Create().
			SetRequestID(requestID).
			SetFieldKind(fieldKindURLEncoded).
			SetKey(k).
			SetValue(v).
			SetEnabled(true).
			SetSortOrder(sortOrder).
			Save(ctx)
		if err != nil {
			return err
		}
		sortOrder++
	}
	for _, p := range in.MultipartParts {
		k := strings.TrimSpace(p.Key)
		if k == "" {
			continue
		}
		kin := strings.ToLower(strings.TrimSpace(p.Kind))
		if kin == "file" {
			fp := strings.TrimSpace(p.FilePath)
			if fp == "" {
				continue
			}
			_, err := tx.RequestFormField.Create().
				SetRequestID(requestID).
				SetFieldKind(fieldKindMultipartFile).
				SetKey(k).
				SetValue(fp).
				SetEnabled(true).
				SetSortOrder(sortOrder).
				Save(ctx)
			if err != nil {
				return err
			}
		} else {
			_, err := tx.RequestFormField.Create().
				SetRequestID(requestID).
				SetFieldKind(fieldKindMultipartText).
				SetKey(k).
				SetValue(strings.TrimSpace(p.Value)).
				SetEnabled(true).
				SetSortOrder(sortOrder).
				Save(ctx)
			if err != nil {
				return err
			}
		}
		sortOrder++
	}
	return nil
}
