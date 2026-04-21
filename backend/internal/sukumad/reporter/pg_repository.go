package reporter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

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

func NewPgRepository(db *sqlx.DB) *PgRepository {
	return &PgRepository{db: db}
}

func (r *PgRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	result := ListResult{Page: query.Page, PageSize: query.PageSize}
	where := "WHERE 1=1"
	args := []interface{}{}
	if strings.TrimSpace(query.Search) != "" {
		where += " AND (LOWER(telephone) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?) OR LOWER(rapidpro_uuid) LIKE LOWER(?))"
		s := fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Search))
		args = append(args, s, s, s)
	}
	if query.OrgUnitID != nil {
		where += " AND org_unit_id = ?"
		args = append(args, *query.OrgUnitID)
	}
	if query.OnlyActive {
		where += " AND is_active = TRUE"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reporters %s", where)
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
		SELECT id, uid, name, telephone, whatsapp, telegram, org_unit_id, reporting_location,
		       district_id, total_reports, last_reporting_date, sms_code, sms_code_expires_at,
		       mtuuid, synced, rapidpro_uuid, is_active, created_at, updated_at, last_login_at
		FROM reporters %s
		ORDER BY name ASC, id ASC
		LIMIT %d OFFSET %d
	`, where, limit, offset)
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

func (r *PgRepository) GetByID(ctx context.Context, id int64) (Reporter, error) {
	return r.getByWhere(ctx, "id = $1", id)
}

func (r *PgRepository) GetByUID(ctx context.Context, uid string) (Reporter, error) {
	return r.getByWhere(ctx, "uid = $1", uid)
}

func (r *PgRepository) GetByContactUUID(ctx context.Context, contactUUID string) (Reporter, error) {
	return r.getByWhere(ctx, "rapidpro_uuid = $1", contactUUID)
}

func (r *PgRepository) GetByPhoneNumber(ctx context.Context, phone string) (Reporter, error) {
	return r.getByWhere(ctx, "telephone = $1", phone)
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
			uid, contact_uuid, phone_number, display_name, org_unit_id, is_active, created_at, updated_at,
			name, telephone, whatsapp, telegram, reporting_location, district_id, total_reports,
			last_reporting_date, sms_code, sms_code_expires_at, mtuuid, synced, rapidpro_uuid, last_login_at
		)
		VALUES (
			COALESCE(NULLIF($1, ''), gen_random_uuid()::text), $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22
		)
		RETURNING id, uid
	`
	if err := tx.QueryRowxContext(
		ctx,
		query,
		reporter.UID,
		reporter.RapidProUUID,
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
		reporter.RapidProUUID,
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
		SET contact_uuid = $1,
		    phone_number = $2,
		    display_name = $3,
		    org_unit_id = $4,
		    is_active = $5,
		    updated_at = $6,
		    name = $7,
		    telephone = $8,
		    whatsapp = $9,
		    telegram = $10,
		    reporting_location = $11,
		    district_id = $12,
		    total_reports = $13,
		    last_reporting_date = $14,
		    sms_code = $15,
		    sms_code_expires_at = $16,
		    mtuuid = $17,
		    synced = $18,
		    rapidpro_uuid = $19,
		    last_login_at = $20
		WHERE id = $21
	`,
		reporter.RapidProUUID,
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
		reporter.RapidProUUID,
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
