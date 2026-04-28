package entryhandler

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"time"

	repo "mediahub_oss/internal/repository"
)

// Early validations of inputs and then splits into either async handling of large file or sync handling of small file
// returns the (partial) entry, a flag if the entry was treated synchronously (true) or asynchronously (false)
// and a possible error
func (h *EntryHandler) createEntryWithFile(ctx context.Context, db repo.Database, entry_request PostPatchEntryRequest, file multipart.File, header *multipart.FileHeader) (EntryWithID, bool, error) {

	var err error

	// create a conversion plan
	procPlan, err := determineConversionPlan(h.MediaConverter, db, header, entry_request.FileName)
	if procPlan.WantsConversion && (procPlan.NeedsConversion && !procPlan.CanConvert) {
		return PartialEntryResponse{}, true, fmt.Errorf("cannot convert %v to the database mime type %v", procPlan.InitMimeType, db.Config.AutoConversion)
	}
	if err != nil {
		return PartialEntryResponse{}, true, err
	}

	// Check if the underlying file is an *os.File. This indicates it's
	// a large file that the http server has spooled to disk.
	if f, ok := file.(*os.File); ok {
		// Path A: Large File, Async
		h.Logger.Debug("CreateEntry: Detected large file. Routing to async handler.")
		partial_resp, err := h.handleLargeFileAsync(ctx, f, db, entry_request, procPlan)
		if err != nil {
			return partial_resp, false, err
		}
		return partial_resp, false, nil
	}

	// Path B: Small File, Sync
	h.Logger.Debug("CreateEntry: Detected small file (in memory). Routing to sync handler.")
	resp, err := h.handleSmallFileSync(ctx, file, db, entry_request, procPlan)
	if err != nil {
		return resp, true, err
	}
	return resp, true, nil

}

// Create a conversion plan and a database entry that contains already the planned filename and mimetype
func (h *EntryHandler) createPreliminaryEntry(ctx context.Context, db repo.Database, entryMetadata PostPatchEntryRequest, plan ProcessingPlan) (repo.Entry, error) {
	var err error

	// size and media_fields are not known yet
	partialEntry := repo.Entry{}
	partialEntry.FileName = plan.FinalFileName
	partialEntry.Timestamp = time.UnixMilli(entryMetadata.Timestamp)
	partialEntry.MimeType = plan.ResultMimeType
	partialEntry.Status = repo.EntryStatusProcessing // Set to processing initially

	// we need to create default values here to not validate the NOT NULL requirement
	partialEntry.MediaFields, err = defaultMediaFields(db.ContentType)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to create default media fields: %w", err)
	}

	partialEntry.CustomFields = entryMetadata.CustomFields

	// 3. Create entry in DB FIRST to get ID and Timestamp
	createdEntry, err := h.Repo.CreateEntry(ctx, db, partialEntry)
	if err != nil {
		return repo.Entry{}, fmt.Errorf("failed to create partial database entry: %w", err)
	}

	return createdEntry, nil
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
