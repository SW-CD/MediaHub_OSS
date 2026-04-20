package audit

import (
	"context"
	"mediahub_oss/internal/repository"
)

type AlDatabase struct {
	Repo repository.Repository
}

func NewAlDatabase(repo repository.Repository) *AlDatabase {
	return &AlDatabase{Repo: repo}
}

func (a *AlDatabase) Log(ctx context.Context, action string, actor string, resource string, details map[string]any) {
	log := repository.AuditLog{
		Action:   action,
		Actor:    actor,
		Resource: resource,
		Details:  details,
	}
	a.Repo.LogAudit(ctx, log)
}
