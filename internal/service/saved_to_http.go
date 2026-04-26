package service

import "PostmanJanai/internal/entity"

// SavedRequestToHTTPInput materialises a SavedRequestFull into an HTTPExecuteInput
// in the same shape the request panel would have built (Phase 8 — used by the runner).
//
// The function clones slices/pointers so callers can mutate the result freely.
func SavedRequestToHTTPInput(full *entity.SavedRequestFull, rootFolderID *string) *entity.HTTPExecuteInput {
	if full == nil {
		return nil
	}
	in := &entity.HTTPExecuteInput{
		Method:             full.Method,
		URL:                full.URL,
		BodyMode:           full.BodyMode,
		Headers:            append([]entity.KeyValue(nil), full.Headers...),
		QueryParams:        append([]entity.KeyValue(nil), full.QueryParams...),
		FormFields:         append([]entity.KeyValue(nil), full.FormFields...),
		MultipartParts:     append([]entity.MultipartPart(nil), full.MultipartParts...),
		InsecureSkipVerify: full.InsecureSkipVerify,
	}
	if full.RawBody != nil {
		in.Body = *full.RawBody
	}
	if full.Auth != nil {
		auth := *full.Auth
		in.Auth = &auth
	}
	id := full.ID
	if id != "" {
		in.RequestID = &id
	}
	if rootFolderID != nil {
		s := *rootFolderID
		if s != "" {
			in.RootFolderID = &s
		}
	}
	return in
}
