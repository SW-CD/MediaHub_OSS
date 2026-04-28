package databasehandler

import (
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"
	"time"
)

// toModel parses the string-based API payload into the Repository model
func (dbc DatabaseCreatePayload) toModel() repository.Database {

	// convert from package internal model to repository model
	customFields := make([]repository.CustomField, len(dbc.CustomFields))
	for i, cf := range dbc.CustomFields {
		customFields[i] = cf.toModel()
	}

	// create return object (ID will be generated automatically by the repository)
	return repository.Database{
		Name:        dbc.Name,
		ContentType: dbc.ContentType,
		Config: repository.DatabaseConfig{
			CreatePreview:  dbc.Config.CreatePreview,
			AutoConversion: dbc.Config.AutoConversion,
		},
		Housekeeping: dbc.Housekeeping.toModel(),
		CustomFields: customFields,
		Stats: repository.DatabaseStats{
			EntryCount:          0,
			TotalDiskSpaceBytes: 0,
		},
	}
}

func (cf DatabaseCustomField) toModel() repository.CustomField {
	return repository.CustomField{
		Name: cf.Name,
		Type: cf.Type,
	}
}

// Extract the config part from the payload and return the repository type
func (upd DatabaseUpdatePayload) getConfig() repository.DatabaseConfig {
	return repository.DatabaseConfig{
		CreatePreview:  upd.Config.CreatePreview,
		AutoConversion: upd.Config.AutoConversion,
	}
}

// Extract the housekeeping part from the payload and return the repository type
func (upd DatabaseUpdatePayload) getHK(lastHKRun time.Time) repository.DatabaseHK {
	hk := upd.Housekeeping.toModel()
	hk.LastHkRun = lastHKRun
	return hk
}

// toModel parses the string-based API payload into the uint64-based Repository model, applying defaults.
func (hk HousekeepingPayload) toModel() repository.DatabaseHK {
	var dbHk repository.DatabaseHK

	// Default: "1h"
	intervalStr := hk.Interval
	if intervalStr == "" {
		intervalStr = "1h"
	}
	if dur, err := shared.ParseDuration(intervalStr); err == nil {
		dbHk.Interval = dur
	}

	// Default: "100G"
	diskSpaceStr := hk.DiskSpace
	if diskSpaceStr == "" {
		diskSpaceStr = "100G"
	}
	if size, err := shared.ParseSize(diskSpaceStr); err == nil {
		dbHk.DiskSpace = size
	}

	// Default: "365d"
	maxAgeStr := hk.MaxAge
	if maxAgeStr == "" {
		maxAgeStr = "365d"
	}
	if age, err := shared.ParseDuration(maxAgeStr); err == nil {
		dbHk.MaxAge = age
	}

	return dbHk
}

func mapToDatabaseResponse(db repository.Database) DatabaseResponse {

	// convert from repository model to package internal model
	customFields := make([]DatabaseCustomField, len(db.CustomFields))
	for i, cf := range db.CustomFields {
		customFields[i] = DatabaseCustomField{
			Name: cf.Name,
			Type: cf.Type,
		}
	}

	// create return object
	return DatabaseResponse{
		ID:          db.ID,
		Name:        db.Name,
		ContentType: db.ContentType,
		Config: ConfigPayload{
			CreatePreview:  db.Config.CreatePreview,
			AutoConversion: db.Config.AutoConversion,
		},
		Housekeeping: DatabaseResponseHK{
			Interval:  shared.DurationToString(db.Housekeeping.Interval),
			DiskSpace: shared.BytesToString(db.Housekeeping.DiskSpace),
			MaxAge:    shared.DurationToString(db.Housekeeping.MaxAge),
		},
		CustomFields: customFields,
		Stats: DatabaseResponseStats{
			EntryCount:          db.Stats.EntryCount,
			TotalDiskSpaceBytes: db.Stats.TotalDiskSpaceBytes,
		},
	}
}
