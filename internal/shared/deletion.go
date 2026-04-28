package shared

import (
	"context"
	"errors"
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/storage"
)

// DeleteSafe safely deletes a single entry from the DB and storage using a 2-Phase approach.
// Returns the entry data of the deleted file and any error if encountered.
func DeleteSafe(ctx context.Context, repo repository.Repository, storage storage.StorageProvider, dbName string, id int64) (repository.DeletedEntryMeta, error) {

	// PHASE 1: LOCK
	// Mark as "Deleting" so it disappears from normal API usage
	if err := repo.UpdateEntriesStatus(ctx, dbName, []int64{id}, repository.EntryStatusDeleting); err != nil {
		return repository.DeletedEntryMeta{}, err // Abort early; database untouched, files untouched!
	}

	// PHASE 2: STORAGE DELETION
	// Attempt to delete the main file from storage
	err := storage.Delete(ctx, dbName, id)

	if err != nil {
		// PHASE 3: ROLLBACK
		// Storage deletion failed, revert stuck file to Error status so admins can investigate
		_ = repo.UpdateEntriesStatus(ctx, dbName, []int64{id}, repository.EntryStatusError)

		return repository.DeletedEntryMeta{}, err
	}

	// We only try to delete the preview if the main file deletion succeeded
	_ = storage.DeletePreview(ctx, dbName, id)

	// PHASE 3: COMMIT
	// Hard delete the record that was successfully wiped from disk
	deletedMeta, deleteErr := repo.DeleteEntry(ctx, dbName, id)
	if deleteErr != nil {
		return repository.DeletedEntryMeta{}, deleteErr
	}
	return deletedMeta, nil

}

// Function to delete files with database entries in a 2-phase approach, to avoid discrepancies
// between the database and the storage.
// Returns
// - entry data of deleted files
// - error if any
func DeleteMultipleSafe(ctx context.Context, repo repository.Repository, storage storage.StorageProvider, dbID string, ids []int64) ([]repository.DeletedEntryMeta, error) {

	// PHASE 1: LOCK
	// Mark as "Deleting" so they disappear from normal API usage
	if err := repo.UpdateEntriesStatus(ctx, dbID, ids, repository.EntryStatusDeleting); err != nil {
		return make([]repository.DeletedEntryMeta, 0), err // Abort early; database untouched, files untouched!
	}

	// PHASE 2: STORAGE DELETION
	delResult, err := storage.DeleteMultiple(ctx, dbID, ids)

	// We only try to delete previews for the files where the main file deletion succeeded
	if len(delResult.Success) > 0 {
		_, _ = storage.DeleteMultiplePreviews(ctx, dbID, delResult.Success)
	}

	// PHASE 3: COMMIT OR ROLLBACK
	var deletedMeta []repository.DeletedEntryMeta
	var newerr error

	// Commit: Hard delete the records that were successfully wiped from disk
	if len(delResult.Success) > 0 {
		deletedMeta, newerr = repo.DeleteEntries(ctx, dbID, delResult.Success)
		err = errors.Join(err, newerr)
	}

	// Rollback: Revert stuck files to Error status so admins can investigate
	if len(delResult.Failed) > 0 {
		_ = repo.UpdateEntriesStatus(ctx, dbID, delResult.Failed, repository.EntryStatusError)
	}

	return deletedMeta, err
}
