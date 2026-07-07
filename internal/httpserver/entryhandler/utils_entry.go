package entryhandler

import (
	repo "mediahub_oss/internal/repository"
)

func mapToPartialEntryResponse(db_id string, entry repo.Entry) PartialEntryResponse {
	statusStr := repo.GetEntryStatusString(entry.Status)

	return PartialEntryResponse{
		DatabaseID:   db_id,
		EntryID:      entry.ID,
		Status:       statusStr,
		Timestamp:    entry.Timestamp.UnixMilli(),
		CreatedAt:    entry.CreatedAt.UnixMilli(),
		UpdatedAt:    entry.UpdatedAt.UnixMilli(),
		MimeType:     entry.MimeType,
		CustomFields: entry.CustomFields,
	}
}

// Helper to map DB Entry to API Response
func mapToEntryResponse(db_id string, entry repo.Entry) EntryResponse {
	statusStr := repo.GetEntryStatusString(entry.Status)

	return EntryResponse{
		DatabaseID:   db_id,
		EntryID:      entry.ID,
		FileName:     entry.FileName,
		Size:         entry.Size,
		PreviewSize:  entry.PreviewSize,
		Status:       statusStr,
		Timestamp:    entry.Timestamp.UnixMilli(),
		CreatedAt:    entry.CreatedAt.UnixMilli(),
		UpdatedAt:    entry.UpdatedAt.UnixMilli(),
		MimeType:     entry.MimeType,
		MediaFields:  entry.MediaFields,
		CustomFields: entry.CustomFields,
	}
}

func (p SearchRequestPayload) toModel() repo.SearchRequest {
	req := repo.SearchRequest{
		Pagination: repo.Pagination{
			Offset: p.Pagination.Offset,
			Limit:  p.Pagination.Limit,
		},
	}

	// Map the Filter if it exists
	if p.Filter != nil {
		var conditions []repo.Condition
		for _, c := range p.Filter.Conditions {
			conditions = append(conditions, repo.Condition{
				Field:    c.Field,
				Operator: c.Operator,
				Value:    c.Value,
			})
		}

		req.Filter = &repo.FilterGroup{
			Operator:   p.Filter.Operator,
			Conditions: conditions,
		}
	}

	// Map the Sort criteria if it exists
	if p.Sort != nil {
		req.Sort = &repo.SortCriteria{
			Field:     p.Sort.Field,
			Direction: p.Sort.Direction,
		}
	}

	return req
}
