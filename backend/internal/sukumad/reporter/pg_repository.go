package reporter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNoEligibleBroadcast = errors.New("no eligible jurisdiction broadcast")

type PgRepository struct {
	db *sqlx.DB
}

type reporterRow struct {
	ID                int64        `db:"id"`
	UID               string       `db:"uid"`
	Name              string       `db:"name"`
	Telephone         string       `db:"telephone"`
	WhatsApp          string       `db:"whatsapp"`
	Telegram          string       `db:"telegram"`
	OrgUnitID         int64        `db:"org_unit_id"`
	ReportingLocation string       `db:"reporting_location"`
	DistrictID        *int64       `db:"district_id"`
	TotalReports      int          `db:"total_reports"`
	LastReportingDate sql.NullTime `db:"last_reporting_date"`
	SMSCode           string       `db:"sms_code"`
	SMSCodeExpiresAt  sql.NullTime `db:"sms_code_expires_at"`
	MTUUID            string       `db:"mtuuid"`
	Synced            bool         `db:"synced"`
	RapidProUUID      string       `db:"rapidpro_uuid"`
	IsActive          bool         `db:"is_active"`
	CreatedAt         time.Time    `db:"created_at"`
	UpdatedAt         time.Time    `db:"updated_at"`
	LastLoginAt       sql.NullTime `db:"last_login_at"`
}

type jurisdictionBroadcastRow struct {
	ID                   int64        `db:"id"`
	UID                  string       `db:"uid"`
	RequestedByUserID    int64        `db:"requested_by_user_id"`
	OrgUnitIDs           []byte       `db:"org_unit_ids"`
	ReporterGroup        string       `db:"reporter_group"`
	MessageText          string       `db:"message_text"`
	DedupeKey            string       `db:"dedupe_key"`
	MatchedCount         int          `db:"matched_count"`
	SentCount            int          `db:"sent_count"`
	FailedCount          int          `db:"failed_count"`
	Status               string       `db:"status"`
	LastError            string       `db:"last_error"`
	RequestedAt          time.Time    `db:"requested_at"`
	StartedAt            sql.NullTime `db:"started_at"`
	FinishedAt           sql.NullTime `db:"finished_at"`
	ClaimedAt            sql.NullTime `db:"claimed_at"`
	ClaimedByWorkerRunID *int64       `db:"claimed_by_worker_run_id"`
	CreatedAt            time.Time    `db:"created_at"`
	UpdatedAt            time.Time    `db:"updated_at"`
}

func NewPgRepository(db *sqlx.DB) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	result := ListResult{Page: query.Page, PageSize: query.PageSize}
	where := "WHERE 1=1"
	args := []interface{}{}
	from := "FROM reporters JOIN org_units scope_org_unit ON scope_org_unit.id = reporters.org_unit_id"

	if query.ScopeRestricted && len(query.ScopePaths) == 0 {
		result.Total = 0
		result.Items = []Reporter{}
		return result, nil
	}
	if strings.TrimSpace(query.Search) != "" {
		where += " AND (LOWER(reporters.telephone) LIKE LOWER(?) OR LOWER(reporters.name) LIKE LOWER(?) OR LOWER(reporters.rapidpro_uuid) LIKE LOWER(?))"
		s := fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Search))
		args = append(args, s, s, s)
	}
	if query.OrgUnitID != nil {
		where += " AND reporters.org_unit_id = ?"
		args = append(args, *query.OrgUnitID)
	}
	if query.OnlyActive {
		where += " AND reporters.is_active = TRUE"
	}
	if query.ScopeRestricted && len(query.ScopePaths) > 0 {
		pathClauses := make([]string, 0, len(query.ScopePaths))
		for _, path := range query.ScopePaths {
			pathClauses = append(pathClauses, "scope_org_unit.path LIKE ?")
			args = append(args, path+"%")
		}
		where += " AND (" + strings.Join(pathClauses, " OR ") + ")"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) %s %s", from, where)
	var total int
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), args...); err != nil {
		return result, err
	}
	result.Total = total

	limit := query.PageSize
	if limit <= 0 {
		limit = 20
	}
	offset := query.Page * limit
	listQuery := fmt.Sprintf(`
		SELECT reporters.id, reporters.uid, reporters.name, reporters.telephone, reporters.whatsapp, reporters.telegram, reporters.org_unit_id, reporters.reporting_location,
		       reporters.district_id, reporters.total_reports, reporters.last_reporting_date, reporters.sms_code, reporters.sms_code_expires_at,
		       reporters.mtuuid, reporters.synced, reporters.rapidpro_uuid, reporters.is_active, reporters.created_at, reporters.updated_at, reporters.last_login_at
		%s
		%s
		ORDER BY reporters.name ASC, reporters.id ASC
		LIMIT %d OFFSET %d
	`, from, where, limit, offset)
	rows := []reporterRow{}
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(listQuery), args...); err != nil {
		return result, err
	}
	items := convertReporterRows(rows)
	if err := r.hydrateGroups(ctx, items); err != nil {
		return result, err
	}
	result.Items = items
	return result, nil
}

func (r *PgRepository) ListBroadcasts(ctx context.Context, query BroadcastListQuery) (BroadcastListResult, error) {
	result := BroadcastListResult{Page: query.Page, PageSize: query.PageSize}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM reporter_broadcasts`); err != nil {
		return result, fmt.Errorf("count reporter broadcasts: %w", err)
	}
	result.Total = total

	limit := query.PageSize
	if limit <= 0 {
		limit = 10
	}
	offset := query.Page * limit
	rows := []jurisdictionBroadcastRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT id, uid, requested_by_user_id, org_unit_ids, reporter_group, message_text, dedupe_key,
		       matched_count, sent_count, failed_count, status, last_error, requested_at, started_at,
		       finished_at, claimed_at, claimed_by_worker_run_id, created_at, updated_at
		FROM reporter_broadcasts
		ORDER BY requested_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset); err != nil {
		return result, fmt.Errorf("list reporter broadcasts: %w", err)
	}

	items := make([]JurisdictionBroadcastRecord, 0, len(rows))
	for _, row := range rows {
		record, err := row.toRecord()
		if err != nil {
			return result, err
		}
		items = append(items, record)
	}
	result.Items = items
	return result, nil
}

func (r *PgRepository) GetByID(ctx context.Context, id int64) (Reporter, error) {
	return r.getByWhere(ctx, "id = $1", id)
}

func (r *PgRepository) GetByUID(ctx context.Context, uid string) (Reporter, error) {
	return r.getByWhere(ctx, "uid = $1", uid)
}

func (r *PgRepository) GetByRapidProUUID(ctx context.Context, rapidProUUID string) (Reporter, error) {
	return r.getByWhere(ctx, "rapidpro_uuid = $1", rapidProUUID)
}

func (r *PgRepository) GetByPhoneNumber(ctx context.Context, phone string) (Reporter, error) {
	return r.getByWhere(ctx, "telephone = $1", phone)
}

func (r *PgRepository) ListByIDs(ctx context.Context, ids []int64) ([]Reporter, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`
		SELECT id, uid, name, telephone, whatsapp, telegram, org_unit_id, reporting_location,
		       district_id, total_reports, last_reporting_date, sms_code, sms_code_expires_at,
		       mtuuid, synced, rapidpro_uuid, is_active, created_at, updated_at, last_login_at
		FROM reporters
		WHERE id IN (?)
		ORDER BY name ASC, id ASC
	`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	rows := []reporterRow{}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	items := convertReporterRows(rows)
	if err := r.hydrateGroups(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PgRepository) ListUpdatedSince(ctx context.Context, since *time.Time, limit int, onlyActive bool) ([]Reporter, error) {
	if limit <= 0 {
		limit = 100
	}
	whereParts := []string{}
	args := make([]any, 0, 2)
	if since != nil {
		whereParts = append(whereParts, "updated_at > ?")
		args = append(args, since.UTC())
	}
	if onlyActive {
		whereParts = append(whereParts, "is_active = TRUE")
	}
	whereParts = append(whereParts, "COALESCE(rapidpro_uuid, '') <> ''")
	where := ""
	if len(whereParts) > 0 {
		where = "WHERE " + strings.Join(whereParts, " AND ")
	}
	query := fmt.Sprintf(`
		SELECT id, uid, name, telephone, whatsapp, telegram, org_unit_id, reporting_location,
		       district_id, total_reports, last_reporting_date, sms_code, sms_code_expires_at,
		       mtuuid, synced, rapidpro_uuid, is_active, created_at, updated_at, last_login_at
		FROM reporters
		%s
		ORDER BY updated_at ASC, id ASC
		LIMIT %d
	`, where, limit)
	rows := []reporterRow{}
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(query), args...); err != nil {
		return nil, err
	}
	items := convertReporterRows(rows)
	if err := r.hydrateGroups(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PgRepository) CountBroadcastRecipients(ctx context.Context, query BroadcastRecipientQuery) (int, error) {
	where, args := buildBroadcastRecipientWhere(query)
	if where == "" {
		return 0, nil
	}
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT reporters.id)
		FROM reporters
		JOIN org_units scope_org_unit ON scope_org_unit.id = reporters.org_unit_id
		JOIN reporter_groups rg ON rg.reporter_id = reporters.id
		WHERE ` + where
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), args...); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *PgRepository) ListBroadcastRecipients(ctx context.Context, query BroadcastRecipientQuery) ([]Reporter, error) {
	where, args := buildBroadcastRecipientWhere(query)
	if where == "" {
		return []Reporter{}, nil
	}
	listQuery := `
		SELECT DISTINCT reporters.id, reporters.uid, reporters.name, reporters.telephone, reporters.whatsapp, reporters.telegram, reporters.org_unit_id, reporters.reporting_location,
		       reporters.district_id, reporters.total_reports, reporters.last_reporting_date, reporters.sms_code, reporters.sms_code_expires_at,
		       reporters.mtuuid, reporters.synced, reporters.rapidpro_uuid, reporters.is_active, reporters.created_at, reporters.updated_at, reporters.last_login_at
		FROM reporters
		JOIN org_units scope_org_unit ON scope_org_unit.id = reporters.org_unit_id
		JOIN reporter_groups rg ON rg.reporter_id = reporters.id
		WHERE ` + where + `
		ORDER BY reporters.name ASC, reporters.id ASC
	`
	rows := []reporterRow{}
	if err := r.db.SelectContext(ctx, &rows, r.db.Rebind(listQuery), args...); err != nil {
		return nil, err
	}
	items := convertReporterRows(rows)
	if err := r.hydrateGroups(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PgRepository) GetRecentPendingBroadcastByDedupeKey(ctx context.Context, dedupeKey string, since time.Time) (JurisdictionBroadcastRecord, error) {
	row := jurisdictionBroadcastRow{}
	if err := r.db.GetContext(ctx, &row, `
		SELECT id, uid, requested_by_user_id, org_unit_ids, reporter_group, message_text, dedupe_key,
		       matched_count, sent_count, failed_count, status, last_error, requested_at, started_at,
		       finished_at, claimed_at, claimed_by_worker_run_id, created_at, updated_at
		FROM reporter_broadcasts
		WHERE dedupe_key = $1
		  AND requested_at >= $2
		  AND status IN ('queued', 'running')
		ORDER BY requested_at DESC, id DESC
		LIMIT 1
	`, strings.TrimSpace(dedupeKey), since.UTC()); err != nil {
		return JurisdictionBroadcastRecord{}, err
	}
	return row.toRecord()
}

func (r *PgRepository) CreateJurisdictionBroadcast(ctx context.Context, record JurisdictionBroadcastRecord) (JurisdictionBroadcastRecord, error) {
	orgUnitIDs, err := marshalInt64List(record.OrgUnitIDs)
	if err != nil {
		return JurisdictionBroadcastRecord{}, err
	}
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO reporter_broadcasts (
			uid, requested_by_user_id, org_unit_ids, reporter_group, message_text, dedupe_key,
			matched_count, sent_count, failed_count, status, last_error, requested_at,
			started_at, finished_at, claimed_at, claimed_by_worker_run_id, created_at, updated_at
		)
		VALUES (
			$1, $2, $3::jsonb, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			NULL, NULL, NULL, NULL, $13, $14
		)
		RETURNING id
	`,
		record.UID,
		record.RequestedByUserID,
		string(orgUnitIDs),
		record.ReporterGroup,
		record.MessageText,
		record.DedupeKey,
		record.MatchedCount,
		record.SentCount,
		record.FailedCount,
		record.Status,
		record.LastError,
		record.RequestedAt.UTC(),
		record.CreatedAt.UTC(),
		record.UpdatedAt.UTC(),
	); err != nil {
		return JurisdictionBroadcastRecord{}, fmt.Errorf("create reporter broadcast: %w", err)
	}
	return r.getJurisdictionBroadcastByID(ctx, id)
}

func (r *PgRepository) ClaimNextJurisdictionBroadcast(ctx context.Context, now time.Time, claimTimeout time.Duration, workerRunID int64) (JurisdictionBroadcastRecord, error) {
	staleBefore := now.UTC().Add(-claimTimeout)
	row := jurisdictionBroadcastRow{}
	if err := r.db.GetContext(ctx, &row, `
		WITH candidate AS (
			SELECT rb.id
			FROM reporter_broadcasts rb
			WHERE rb.status = 'queued'
			  AND (rb.claimed_at IS NULL OR rb.claimed_at <= $2)
			ORDER BY rb.requested_at ASC, rb.id ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		), claimed AS (
			UPDATE reporter_broadcasts rb
			SET status = 'running',
			    started_at = COALESCE(rb.started_at, $1),
			    claimed_at = $1,
			    claimed_by_worker_run_id = NULLIF($3, 0),
			    updated_at = NOW()
			FROM candidate
			WHERE rb.id = candidate.id
			RETURNING rb.id
		)
		SELECT id, uid, requested_by_user_id, org_unit_ids, reporter_group, message_text, dedupe_key,
		       matched_count, sent_count, failed_count, status, last_error, requested_at, started_at,
		       finished_at, claimed_at, claimed_by_worker_run_id, created_at, updated_at
		FROM reporter_broadcasts
		JOIN claimed USING (id)
	`, now.UTC(), staleBefore, workerRunID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return JurisdictionBroadcastRecord{}, ErrNoEligibleBroadcast
		}
		return JurisdictionBroadcastRecord{}, fmt.Errorf("claim reporter broadcast: %w", err)
	}
	return row.toRecord()
}

func (r *PgRepository) UpdateJurisdictionBroadcastResult(ctx context.Context, id int64, status string, sentCount int, failedCount int, lastError string, finishedAt time.Time) (JurisdictionBroadcastRecord, error) {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE reporter_broadcasts
		SET status = $2,
		    sent_count = $3,
		    failed_count = $4,
		    last_error = $5,
		    finished_at = $6,
		    claimed_at = NULL,
		    claimed_by_worker_run_id = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, id, strings.TrimSpace(status), sentCount, failedCount, strings.TrimSpace(lastError), finishedAt.UTC()); err != nil {
		return JurisdictionBroadcastRecord{}, fmt.Errorf("update reporter broadcast: %w", err)
	}
	return r.getJurisdictionBroadcastByID(ctx, id)
}

func (r *PgRepository) UpdateRapidProStatus(ctx context.Context, id int64, rapidProUUID string, synced bool) (Reporter, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE reporters
		SET rapidpro_uuid = $2,
		    synced = $3
		WHERE id = $1
	`, id, strings.TrimSpace(rapidProUUID), synced)
	if err != nil {
		return Reporter{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return Reporter{}, err
	}
	if rows == 0 {
		return Reporter{}, sql.ErrNoRows
	}
	return r.GetByID(ctx, id)
}

func (r *PgRepository) Create(ctx context.Context, reporter Reporter) (Reporter, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Reporter{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	location, districtID, err := r.resolveLocationFields(ctx, tx, reporter.OrgUnitID)
	if err != nil {
		return Reporter{}, err
	}
	reporter.ReportingLocation = location
	reporter.DistrictID = districtID

	query := `
		INSERT INTO reporters (
			uid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at,
			name, telephone, whatsapp, telegram, reporting_location, district_id, total_reports,
			last_reporting_date, sms_code, sms_code_expires_at, mtuuid, synced, rapidpro_uuid, last_login_at
		)
		VALUES (
			COALESCE(NULLIF($1, ''), gen_random_uuid()::text), $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21
		)
		RETURNING id, uid
	`
	if err := tx.QueryRowxContext(
		ctx,
		query,
		reporter.UID,
		reporter.Telephone,
		reporter.Name,
		reporter.OrgUnitID,
		reporter.IsActive,
		reporter.CreatedAt,
		reporter.UpdatedAt,
		reporter.Name,
		reporter.Telephone,
		reporter.WhatsApp,
		reporter.Telegram,
		reporter.ReportingLocation,
		reporter.DistrictID,
		reporter.TotalReports,
		reporter.LastReportingDate,
		reporter.SMSCode,
		reporter.SMSCodeExpiresAt,
		reporter.MTUUID,
		reporter.Synced,
		strings.TrimSpace(reporter.RapidProUUID),
		reporter.LastLoginAt,
	).Scan(&reporter.ID, &reporter.UID); err != nil {
		return Reporter{}, err
	}

	if err := r.replaceGroups(ctx, tx, reporter.ID, reporter.Groups); err != nil {
		return Reporter{}, err
	}

	if err := tx.Commit(); err != nil {
		return Reporter{}, err
	}
	tx = nil
	return r.GetByID(ctx, reporter.ID)
}

func (r *PgRepository) Update(ctx context.Context, reporter Reporter) (Reporter, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Reporter{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	existing, err := r.getByIDTx(ctx, tx, reporter.ID)
	if err != nil {
		return Reporter{}, err
	}

	location, districtID, err := r.resolveLocationFields(ctx, tx, reporter.OrgUnitID)
	if err != nil {
		return Reporter{}, err
	}

	reporter = preserveSystemManagedFields(existing, reporter)

	res, err := tx.ExecContext(ctx, `
		UPDATE reporters
		SET phone_number = $1,
		    display_name = $2,
		    org_unit_id = $3,
		    is_active = $4,
		    updated_at = $5,
		    name = $6,
		    telephone = $7,
		    whatsapp = $8,
		    telegram = $9,
		    reporting_location = $10,
		    district_id = $11,
		    total_reports = $12,
		    last_reporting_date = $13,
		    sms_code = $14,
		    sms_code_expires_at = $15,
		    mtuuid = $16,
		    synced = $17,
		    rapidpro_uuid = $18,
		    last_login_at = $19
		WHERE id = $20
	`,
		reporter.Telephone,
		reporter.Name,
		reporter.OrgUnitID,
		reporter.IsActive,
		reporter.UpdatedAt,
		reporter.Name,
		reporter.Telephone,
		reporter.WhatsApp,
		reporter.Telegram,
		location,
		districtID,
		reporter.TotalReports,
		reporter.LastReportingDate,
		reporter.SMSCode,
		reporter.SMSCodeExpiresAt,
		reporter.MTUUID,
		reporter.Synced,
		strings.TrimSpace(reporter.RapidProUUID),
		reporter.LastLoginAt,
		reporter.ID,
	)
	if err != nil {
		return Reporter{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return Reporter{}, err
	}
	if rows == 0 {
		return Reporter{}, sql.ErrNoRows
	}

	if err := r.replaceGroups(ctx, tx, reporter.ID, reporter.Groups); err != nil {
		return Reporter{}, err
	}

	if err := tx.Commit(); err != nil {
		return Reporter{}, err
	}
	tx = nil
	return r.GetByID(ctx, reporter.ID)
}

func preserveSystemManagedFields(existing Reporter, incoming Reporter) Reporter {
	if incoming.UID == "" {
		incoming.UID = existing.UID
	}
	if incoming.TotalReports == 0 && existing.TotalReports != 0 {
		incoming.TotalReports = existing.TotalReports
	}
	if incoming.LastReportingDate == nil {
		incoming.LastReportingDate = existing.LastReportingDate
	}
	if incoming.SMSCode == "" {
		incoming.SMSCode = existing.SMSCode
	}
	if incoming.SMSCodeExpiresAt == nil {
		incoming.SMSCodeExpiresAt = existing.SMSCodeExpiresAt
	}
	if incoming.MTUUID == "" {
		incoming.MTUUID = existing.MTUUID
	}
	if incoming.Synced == false && existing.Synced {
		incoming.Synced = existing.Synced
	}
	if incoming.LastLoginAt == nil {
		incoming.LastLoginAt = existing.LastLoginAt
	}
	return incoming
}

func (r *PgRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM reporters WHERE id = $1", id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PgRepository) getByWhere(ctx context.Context, where string, arg any) (Reporter, error) {
	row := reporterRow{}
	query := `
		SELECT id, uid, name, telephone, whatsapp, telegram, org_unit_id, reporting_location,
		       district_id, total_reports, last_reporting_date, sms_code, sms_code_expires_at,
		       mtuuid, synced, rapidpro_uuid, is_active, created_at, updated_at, last_login_at
		FROM reporters
		WHERE ` + where
	if err := r.db.GetContext(ctx, &row, query, arg); err != nil {
		return Reporter{}, err
	}
	reporter := row.toReporter()
	groups, err := r.listGroupsByReporterID(ctx, reporter.ID)
	if err != nil {
		return Reporter{}, err
	}
	reporter.Groups = groups
	return reporter, nil
}

func (r *PgRepository) getJurisdictionBroadcastByID(ctx context.Context, id int64) (JurisdictionBroadcastRecord, error) {
	row := jurisdictionBroadcastRow{}
	if err := r.db.GetContext(ctx, &row, `
		SELECT id, uid, requested_by_user_id, org_unit_ids, reporter_group, message_text, dedupe_key,
		       matched_count, sent_count, failed_count, status, last_error, requested_at, started_at,
		       finished_at, claimed_at, claimed_by_worker_run_id, created_at, updated_at
		FROM reporter_broadcasts
		WHERE id = $1
	`, id); err != nil {
		return JurisdictionBroadcastRecord{}, err
	}
	return row.toRecord()
}

func (r *PgRepository) getByIDTx(ctx context.Context, tx *sqlx.Tx, id int64) (Reporter, error) {
	row := reporterRow{}
	if err := tx.GetContext(ctx, &row, `
		SELECT id, uid, name, telephone, whatsapp, telegram, org_unit_id, reporting_location,
		       district_id, total_reports, last_reporting_date, sms_code, sms_code_expires_at,
		       mtuuid, synced, rapidpro_uuid, is_active, created_at, updated_at, last_login_at
		FROM reporters
		WHERE id = $1
	`, id); err != nil {
		return Reporter{}, err
	}
	return row.toReporter(), nil
}

func (r *PgRepository) resolveLocationFields(ctx context.Context, tx *sqlx.Tx, orgUnitID int64) (string, *int64, error) {
	type locationRow struct {
		ID             int64  `db:"id"`
		Name           string `db:"name"`
		HierarchyLevel int    `db:"hierarchy_level"`
	}
	var targetPath string
	if err := tx.GetContext(ctx, &targetPath, `SELECT path FROM org_units WHERE id = $1`, orgUnitID); err != nil {
		return "", nil, err
	}
	rows := []locationRow{}
	if err := tx.SelectContext(ctx, &rows, `
		SELECT id, name, hierarchy_level
		FROM org_units
		WHERE $1 LIKE path || '%'
		ORDER BY hierarchy_level ASC, path ASC
	`, targetPath); err != nil {
		return "", nil, err
	}
	if len(rows) == 0 {
		return "", nil, fmt.Errorf("organisation unit %d not found", orgUnitID)
	}
	parts := make([]string, 0, len(rows))
	var districtID *int64
	for _, row := range rows {
		parts = append(parts, row.Name)
		if row.HierarchyLevel == 2 && districtID == nil {
			id := row.ID
			districtID = &id
		}
	}
	if districtID == nil {
		id := rows[0].ID
		districtID = &id
	}
	return strings.Join(parts, " / "), districtID, nil
}

func (r *PgRepository) replaceGroups(ctx context.Context, tx *sqlx.Tx, reporterID int64, groups []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM reporter_groups WHERE reporter_id = $1`, reporterID); err != nil {
		return err
	}
	for _, group := range groups {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reporter_groups (reporter_id, group_name)
			VALUES ($1, $2)
		`, reporterID, group); err != nil {
			return err
		}
	}
	return nil
}

func (r *PgRepository) hydrateGroups(ctx context.Context, reporters []Reporter) error {
	if len(reporters) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(reporters))
	indexByID := make(map[int64]*Reporter, len(reporters))
	for i := range reporters {
		reporters[i].Groups = []string{}
		ids = append(ids, reporters[i].ID)
		indexByID[reporters[i].ID] = &reporters[i]
	}
	query, args, err := sqlx.In(`
		SELECT reporter_id, group_name
		FROM reporter_groups
		WHERE reporter_id IN (?)
		ORDER BY reporter_id ASC, group_name ASC
	`, ids)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)
	type groupRow struct {
		ReporterID int64  `db:"reporter_id"`
		GroupName  string `db:"group_name"`
	}
	rows := []groupRow{}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return err
	}
	for _, row := range rows {
		if reporter, ok := indexByID[row.ReporterID]; ok {
			reporter.Groups = append(reporter.Groups, row.GroupName)
		}
	}
	return nil
}

func (r *PgRepository) listGroupsByReporterID(ctx context.Context, reporterID int64) ([]string, error) {
	values := []string{}
	if err := r.db.SelectContext(ctx, &values, `
		SELECT group_name
		FROM reporter_groups
		WHERE reporter_id = $1
		ORDER BY group_name ASC
	`, reporterID); err != nil {
		return nil, err
	}
	return values, nil
}

func convertReporterRows(rows []reporterRow) []Reporter {
	items := make([]Reporter, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toReporter())
	}
	return items
}

func (r reporterRow) toReporter() Reporter {
	return Reporter{
		ID:                r.ID,
		UID:               r.UID,
		Name:              r.Name,
		Telephone:         r.Telephone,
		WhatsApp:          r.WhatsApp,
		Telegram:          r.Telegram,
		OrgUnitID:         r.OrgUnitID,
		ReportingLocation: r.ReportingLocation,
		DistrictID:        r.DistrictID,
		TotalReports:      r.TotalReports,
		LastReportingDate: nullTimePtr(r.LastReportingDate),
		SMSCode:           r.SMSCode,
		SMSCodeExpiresAt:  nullTimePtr(r.SMSCodeExpiresAt),
		MTUUID:            r.MTUUID,
		Synced:            r.Synced,
		RapidProUUID:      r.RapidProUUID,
		IsActive:          r.IsActive,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		LastLoginAt:       nullTimePtr(r.LastLoginAt),
		Groups:            []string{},
	}
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func buildBroadcastRecipientWhere(query BroadcastRecipientQuery) (string, []any) {
	if len(query.OrgUnitPaths) == 0 || strings.TrimSpace(query.ReporterGroup) == "" {
		return "", nil
	}
	whereParts := []string{"LOWER(rg.group_name) = LOWER(?)"}
	args := []any{strings.TrimSpace(query.ReporterGroup)}
	pathClauses := make([]string, 0, len(query.OrgUnitPaths))
	for _, path := range query.OrgUnitPaths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		pathClauses = append(pathClauses, "scope_org_unit.path LIKE ?")
		args = append(args, trimmed+"%")
	}
	if len(pathClauses) == 0 {
		return "", nil
	}
	whereParts = append(whereParts, "("+strings.Join(pathClauses, " OR ")+")")
	if query.OnlyActive {
		whereParts = append(whereParts, "reporters.is_active = TRUE")
	}
	return strings.Join(whereParts, " AND "), args
}

func marshalInt64List(values []int64) ([]byte, error) {
	return json.Marshal(values)
}

func unmarshalInt64List(raw []byte) ([]int64, error) {
	if len(raw) == 0 {
		return []int64{}, nil
	}
	var values []int64
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("decode org unit ids: %w", err)
	}
	return values, nil
}

func (r jurisdictionBroadcastRow) toRecord() (JurisdictionBroadcastRecord, error) {
	orgUnitIDs, err := unmarshalInt64List(r.OrgUnitIDs)
	if err != nil {
		return JurisdictionBroadcastRecord{}, err
	}
	return JurisdictionBroadcastRecord{
		ID:                   r.ID,
		UID:                  r.UID,
		RequestedByUserID:    r.RequestedByUserID,
		OrgUnitIDs:           orgUnitIDs,
		ReporterGroup:        r.ReporterGroup,
		MessageText:          r.MessageText,
		DedupeKey:            r.DedupeKey,
		MatchedCount:         r.MatchedCount,
		SentCount:            r.SentCount,
		FailedCount:          r.FailedCount,
		Status:               r.Status,
		LastError:            r.LastError,
		RequestedAt:          r.RequestedAt,
		StartedAt:            nullTimePtr(r.StartedAt),
		FinishedAt:           nullTimePtr(r.FinishedAt),
		ClaimedAt:            nullTimePtr(r.ClaimedAt),
		ClaimedByWorkerRunID: r.ClaimedByWorkerRunID,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}, nil
}
