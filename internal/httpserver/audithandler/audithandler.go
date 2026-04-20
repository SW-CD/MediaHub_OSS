package audithandler

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/repository"
)

// @Summary Get audit logs
// @Description Retrieves a paginated list of system audit logs. This endpoint provides basic filtering by timestamp and pagination.
// @Tags audit
// @Produce  json
// @Param   limit   query  int     false  "Number of logs to return (default 30)"
// @Param   offset  query  int     false  "Offset for pagination (default 0)"
// @Param   order   query  string  false  "Sort order ('asc' or 'desc', default 'desc')"
// @Param   tstart  query  int64   false  "Start timestamp (Unix milliseconds)"
// @Param   tend    query  int64   false  "End timestamp (Unix milliseconds)"
// @Success 200 {array} AuditLogResponse "Returns an array of audit log objects"
// @Failure 400 {object} utils.ErrorResponse "Invalid parameter formats"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden (Requires IsAdmin role)"
// @Failure 500 {object} utils.ErrorResponse "Failed to retrieve audit logs"
// @Security BasicAuth
// @Security BearerAuth
// @Router /audit [get]
func (h *AuditHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Extract and validate user
	user := utils.GetUserFromContext(ctx)
	if user == nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "could not get user from context")
		return
	}

	// 2. Parse query parameters safely
	limit := parseQueryInt(r, "limit", 30)
	offset := parseQueryInt(r, "offset", 0)

	order := r.URL.Query().Get("order")
	if order == "" {
		order = "desc"
	}

	var tStart, tEnd time.Time
	tStartInt := parseQueryInt64(r, "tstart", math.MinInt64)
	if tStartInt != math.MinInt64 {
		tStart = time.UnixMilli(tStartInt)
	}
	tEndInt := parseQueryInt64(r, "tend", math.MaxInt64)
	if tEndInt != math.MaxInt64 {
		tEnd = time.UnixMilli(tEndInt)
	}

	// 3. Fetch logs from repository
	logs, err := h.Repo.GetLogs(ctx, limit, offset, order, tStart, tEnd)
	if err != nil {
		h.Logger.Error("Failed to retrieve audit logs", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve audit logs")
		return
	}

	// 4. Map to response DTO (converting time.Time to Unix milliseconds)
	resp := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		resp[i] = newAuditLogResponse(log)
	}

	utils.RespondWithJSON(w, http.StatusOK, resp)
}

// --- Helper functions for parsing query parameters safely ---

func newAuditLogResponse(log repository.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:        log.ID,
		Timestamp: log.Timestamp.UnixMilli(),
		Action:    log.Action,
		Actor:     log.Actor,
		Resource:  log.Resource,
		Details:   log.Details,
	}
}

func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue // Return default if parsing fails instead of throwing a 400
	}
	return val
}

func parseQueryInt64(r *http.Request, key string, defaultValue int64) int64 {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return val
}
