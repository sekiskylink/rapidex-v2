package orgunit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type reporterHierarchySnapshot struct {
	ReporterID  int64
	OrgUnitUID  string
	OrgUnitName string
}

type userOrgUnitSnapshot struct {
	UserID     int64
	OrgUnitUID string
}

func (r *PgRepository) GetSyncState(ctx context.Context) (SyncState, error) {
	state := SyncState{}
	err := r.db.GetContext(ctx, &state, `
		SELECT last_started_at, last_completed_at, last_synced_at, last_status, last_error,
		       source_server_code, district_level_name, district_level_code, last_counts
		FROM org_unit_sync_state
		WHERE id = 1
	`)
	if err != nil {
		return SyncState{}, err
	}
	return state, nil
}

func (r *PgRepository) ReplaceHierarchy(ctx context.Context, input replaceHierarchyInput) (SyncResult, error) {
	result := SyncResult{
		ServerCode:          input.ServerCode,
		DryRun:              input.Request.DryRun,
		FullRefresh:         input.Request.FullRefresh,
		DistrictLevelName:   firstNonEmpty(input.ResolvedLevelName, input.Request.DistrictLevelName),
		DistrictLevelCode:   input.Request.DistrictLevelCode,
		ResolvedDistrictUID: input.ResolvedLevelUID,
		ResolvedDistrict:    input.ResolvedLevelName,
		StartedAt:           input.StartedAt,
		CompletedAt:         input.CompletedAt,
		Status:              input.Status,
		LevelsCount:         len(input.Levels),
		GroupsCount:         len(input.Groups),
		AttributesCount:     len(input.Attributes),
		OrgUnitsCount:       len(input.OrgUnits),
		GroupMembersCount:   groupMemberCount(input.GroupMembers),
		ErrorMessage:        strings.TrimSpace(input.ErrorMessage),
	}
	if input.Request.DryRun || input.Status == syncStatusFailed {
		if err := r.upsertSyncState(ctx, nil, input, result); err != nil {
			return SyncResult{}, err
		}
		return result, nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if input.Request.InitialSync {
		if err := r.replaceHierarchyInitial(ctx, tx, input, &result); err != nil {
			return SyncResult{}, err
		}
	} else {
		if err := r.replaceHierarchyReconcile(ctx, tx, input, &result); err != nil {
			return SyncResult{}, err
		}
	}

	if err := r.upsertSyncState(ctx, tx, input, result); err != nil {
		return SyncResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return SyncResult{}, err
	}
	tx = nil
	return result, nil
}

func (r *PgRepository) replaceHierarchyInitial(ctx context.Context, tx *sqlx.Tx, input replaceHierarchyInput, result *SyncResult) error {
	deletedAssignments, err := countRows(ctx, tx, `SELECT COUNT(*) FROM user_org_units`)
	if err != nil {
		return err
	}
	deletedReporters, err := countRows(ctx, tx, `SELECT COUNT(*) FROM reporters`)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_org_units`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM reporters`); err != nil {
		return err
	}
	if err := r.deleteHierarchyTables(ctx, tx); err != nil {
		return err
	}
	orgUnitIDs, _, err := r.insertHierarchy(ctx, tx, input)
	if err != nil {
		return err
	}
	if err := r.insertGroupMembers(ctx, tx, input.GroupMembers, orgUnitIDs); err != nil {
		return err
	}
	result.DeletedAssignments = deletedAssignments
	result.DeletedReporters = deletedReporters
	return nil
}

func (r *PgRepository) replaceHierarchyReconcile(ctx context.Context, tx *sqlx.Tx, input replaceHierarchyInput, result *SyncResult) error {
	deletedAssignments, err := countRows(ctx, tx, `SELECT COUNT(*) FROM user_org_units`)
	if err != nil {
		return err
	}
	reporterSnapshots, err := snapshotReporters(ctx, tx)
	if err != nil {
		return err
	}
	userSnapshots, err := snapshotUserOrgUnits(ctx, tx)
	if err != nil {
		return err
	}
	if err := detachReporters(ctx, tx, reporterSnapshots, input.StartedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_org_units`); err != nil {
		return err
	}
	if err := r.deleteHierarchyTables(ctx, tx); err != nil {
		return err
	}
	orgUnitIDs, orgUnitNames, err := r.insertHierarchy(ctx, tx, input)
	if err != nil {
		return err
	}
	if err := r.insertGroupMembers(ctx, tx, input.GroupMembers, orgUnitIDs); err != nil {
		return err
	}

	remapped := 0
	orphaned := 0
	for _, item := range reporterSnapshots {
		targetID, ok := orgUnitIDs[item.OrgUnitUID]
		if !ok || strings.TrimSpace(item.OrgUnitUID) == "" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE reporters
				SET org_unit_id = NULL,
				    reporting_location = '',
				    district_id = NULL,
				    orphaned_at = COALESCE(orphaned_at, $2),
				    orphan_reason = 'dhis2_org_unit_missing_after_refresh',
				    last_known_org_unit_uid = $3,
				    last_known_org_unit_name = $4,
				    synced = FALSE,
				    updated_at = NOW()
				WHERE id = $1
			`, item.ReporterID, input.StartedAt, item.OrgUnitUID, item.OrgUnitName); err != nil {
				return err
			}
			orphaned++
			continue
		}
		location, districtID, err := resolveLocationFieldsTx(ctx, tx, targetID)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE reporters
			SET org_unit_id = $2,
			    reporting_location = $3,
			    district_id = $4,
			    orphaned_at = NULL,
			    orphan_reason = '',
			    last_known_org_unit_uid = $5,
			    last_known_org_unit_name = $6,
			    synced = FALSE,
			    updated_at = NOW()
			WHERE id = $1
		`, item.ReporterID, targetID, location, districtID, item.OrgUnitUID, orgUnitNames[item.OrgUnitUID]); err != nil {
			return err
		}
		remapped++
	}

	reassignedAssignments := 0
	droppedAssignments := 0
	for _, item := range userSnapshots {
		targetID, ok := orgUnitIDs[item.OrgUnitUID]
		if !ok || strings.TrimSpace(item.OrgUnitUID) == "" {
			droppedAssignments++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_org_units (user_id, org_unit_id, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (user_id, org_unit_id) DO NOTHING
		`, item.UserID, targetID); err != nil {
			return err
		}
		reassignedAssignments++
	}

	result.DeletedAssignments = deletedAssignments
	result.DeletedReporters = 0
	result.RemappedReporters = remapped
	result.OrphanedReporters = orphaned
	result.ReassignedAssignments = reassignedAssignments
	result.DroppedAssignments = droppedAssignments
	return nil
}

func (r *PgRepository) deleteHierarchyTables(ctx context.Context, tx *sqlx.Tx) error {
	statements := []string{
		`DELETE FROM org_unit_group_members`,
		`DELETE FROM org_unit_attributes`,
		`DELETE FROM org_unit_groups`,
		`DELETE FROM org_unit_levels`,
		`DELETE FROM org_units`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (r *PgRepository) insertHierarchy(ctx context.Context, tx *sqlx.Tx, input replaceHierarchyInput) (map[string]int64, map[string]string, error) {
	groupIDs := make(map[string]int64, len(input.Groups))
	for _, item := range input.Groups {
		var id int64
		if err := tx.QueryRowxContext(ctx, `
			INSERT INTO org_unit_groups (uid, code, name, short_name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id
		`, item.UID, item.Code, item.Name, item.ShortName).Scan(&id); err != nil {
			return nil, nil, err
		}
		groupIDs[item.UID] = id
	}
	for _, item := range input.Levels {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO org_unit_levels (uid, code, name, level, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, item.UID, item.Code, item.Name, item.Level); err != nil {
			return nil, nil, err
		}
	}
	for _, item := range input.Attributes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO org_unit_attributes (uid, code, name, short_name, value_type, is_unique, mandatory, organisation_unit_attribute, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		`, item.UID, item.Code, item.Name, item.ShortName, item.ValueType, item.IsUnique, item.Mandatory, item.OrganisationUnitAttribute); err != nil {
			return nil, nil, err
		}
	}

	orgUnitIDs := make(map[string]int64, len(input.OrgUnits))
	orgUnitNames := make(map[string]string, len(input.OrgUnits))
	syncTime := input.StartedAt
	if input.CompletedAt != nil {
		syncTime = input.CompletedAt.UTC()
	}
	for _, item := range input.OrgUnits {
		var parentID *int64
		if parentUID, _ := item.Extras["parentUid"].(string); strings.TrimSpace(parentUID) != "" {
			id, ok := orgUnitIDs[parentUID]
			if !ok {
				return nil, nil, fmt.Errorf("parent org unit %s not inserted before child %s", parentUID, item.UID)
			}
			parentID = &id
		}
		extras := cloneJSONMap(item.Extras)
		delete(extras, "parentUid")
		var id int64
		if err := tx.QueryRowxContext(ctx, `
			INSERT INTO org_units (
				uid, code, name, short_name, description, parent_id, hierarchy_level, path,
				address, email, url, phone_number, extras, attribute_values, opening_date,
				deleted, last_sync_date, created_at, updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, $12, $13, $14, $15,
				$16, $17, NOW(), NOW()
			)
			RETURNING id
		`,
			item.UID,
			item.Code,
			item.Name,
			item.ShortName,
			item.Description,
			parentID,
			item.HierarchyLevel,
			item.Path,
			item.Address,
			item.Email,
			item.URL,
			item.PhoneNumber,
			extras,
			item.AttributeValues,
			item.OpeningDate,
			item.Deleted,
			syncTime,
		).Scan(&id); err != nil {
			return nil, nil, err
		}
		orgUnitIDs[item.UID] = id
		orgUnitNames[item.UID] = item.Name
	}
	return orgUnitIDs, orgUnitNames, nil
}

func (r *PgRepository) insertGroupMembers(ctx context.Context, tx *sqlx.Tx, groupMembers map[string][]string, orgUnitIDs map[string]int64) error {
	groupIDs := map[string]int64{}
	rows := []struct {
		ID  int64  `db:"id"`
		UID string `db:"uid"`
	}{}
	if err := tx.SelectContext(ctx, &rows, `SELECT id, uid FROM org_unit_groups`); err != nil {
		return err
	}
	for _, row := range rows {
		groupIDs[row.UID] = row.ID
	}
	for orgUnitUID, groupUIDs := range groupMembers {
		orgUnitID, ok := orgUnitIDs[orgUnitUID]
		if !ok {
			return fmt.Errorf("group membership references unknown org unit %s", orgUnitUID)
		}
		for _, groupUID := range groupUIDs {
			groupID, ok := groupIDs[groupUID]
			if !ok {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO org_unit_group_members (org_unit_id, org_unit_group_id, created_at, updated_at)
				VALUES ($1, $2, NOW(), NOW())
			`, orgUnitID, groupID); err != nil {
				return err
			}
		}
	}
	return nil
}

func snapshotReporters(ctx context.Context, tx *sqlx.Tx) ([]reporterHierarchySnapshot, error) {
	rows := []reporterHierarchySnapshot{}
	if err := tx.SelectContext(ctx, &rows, `
		SELECT r.id AS reporter_id, COALESCE(ou.uid, '') AS org_unit_uid, COALESCE(ou.name, '') AS org_unit_name
		FROM reporters r
		LEFT JOIN org_units ou ON ou.id = r.org_unit_id
		ORDER BY r.id ASC
	`); err != nil {
		return nil, err
	}
	return rows, nil
}

func snapshotUserOrgUnits(ctx context.Context, tx *sqlx.Tx) ([]userOrgUnitSnapshot, error) {
	rows := []userOrgUnitSnapshot{}
	if err := tx.SelectContext(ctx, &rows, `
		SELECT u.user_id, COALESCE(ou.uid, '') AS org_unit_uid
		FROM user_org_units u
		JOIN org_units ou ON ou.id = u.org_unit_id
		ORDER BY u.user_id ASC, u.org_unit_id ASC
	`); err != nil {
		return nil, err
	}
	return rows, nil
}

func detachReporters(ctx context.Context, tx *sqlx.Tx, snapshots []reporterHierarchySnapshot, startedAt time.Time) error {
	for _, item := range snapshots {
		if _, err := tx.ExecContext(ctx, `
			UPDATE reporters
			SET org_unit_id = NULL,
			    reporting_location = '',
			    district_id = NULL,
			    orphaned_at = COALESCE(orphaned_at, $2),
			    orphan_reason = CASE
			        WHEN COALESCE(last_known_org_unit_uid, '') = '' THEN 'hierarchy_refresh_reassignment_pending'
			        ELSE orphan_reason
			    END,
			    last_known_org_unit_uid = $3,
			    last_known_org_unit_name = $4,
			    synced = FALSE,
			    updated_at = NOW()
			WHERE id = $1
		`, item.ReporterID, startedAt, item.OrgUnitUID, item.OrgUnitName); err != nil {
			return err
		}
	}
	return nil
}

func resolveLocationFieldsTx(ctx context.Context, tx *sqlx.Tx, orgUnitID int64) (string, *int64, error) {
	type locationRow struct {
		ID             int64  `db:"id"`
		Name           string `db:"name"`
		HierarchyLevel int    `db:"hierarchy_level"`
		LevelName      string `db:"level_name"`
		LevelCode      string `db:"level_code"`
	}
	type districtConfig struct {
		Name string `db:"district_level_name"`
		Code string `db:"district_level_code"`
	}
	var targetPath string
	if err := tx.GetContext(ctx, &targetPath, `SELECT path FROM org_units WHERE id = $1`, orgUnitID); err != nil {
		return "", nil, err
	}
	cfg := districtConfig{}
	_ = tx.GetContext(ctx, &cfg, `SELECT district_level_name, district_level_code FROM org_unit_sync_state WHERE id = 1`)
	rows := []locationRow{}
	if err := tx.SelectContext(ctx, &rows, `
		SELECT ou.id, ou.name, ou.hierarchy_level, COALESCE(l.name, '') AS level_name, COALESCE(l.code, '') AS level_code
		FROM org_units ou
		LEFT JOIN org_unit_levels l ON l.level = ou.hierarchy_level
		WHERE $1 LIKE ou.path || '%'
		ORDER BY ou.hierarchy_level ASC, ou.path ASC
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
		if districtID == nil && ((cfg.Name != "" && strings.EqualFold(strings.TrimSpace(row.LevelName), strings.TrimSpace(cfg.Name))) ||
			(cfg.Code != "" && strings.EqualFold(strings.TrimSpace(row.LevelCode), strings.TrimSpace(cfg.Code))) ||
			(cfg.Name == "" && cfg.Code == "" && row.HierarchyLevel == 2)) {
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

func (r *PgRepository) upsertSyncState(ctx context.Context, tx *sqlx.Tx, input replaceHierarchyInput, result SyncResult) error {
	counts := JSONMap{
		"levelsCount":           result.LevelsCount,
		"groupsCount":           result.GroupsCount,
		"attributesCount":       result.AttributesCount,
		"orgUnitsCount":         result.OrgUnitsCount,
		"groupMembersCount":     result.GroupMembersCount,
		"deletedAssignments":    result.DeletedAssignments,
		"deletedReporters":      result.DeletedReporters,
		"orphanedReporters":     result.OrphanedReporters,
		"remappedReporters":     result.RemappedReporters,
		"reassignedAssignments": result.ReassignedAssignments,
		"droppedAssignments":    result.DroppedAssignments,
		"dryRun":                result.DryRun,
		"initialSync":           input.Request.InitialSync,
	}
	exec := r.db.ExecContext
	if tx != nil {
		exec = tx.ExecContext
	}
	_, err := exec(ctx, `
		INSERT INTO org_unit_sync_state (
			id, last_started_at, last_completed_at, last_synced_at, last_status, last_error,
			source_server_code, district_level_name, district_level_code, last_counts, updated_at
		)
		VALUES (
			1, $1, $2, $3, $4, $5,
			$6, $7, $8, $9, NOW()
		)
		ON CONFLICT (id) DO UPDATE SET
			last_started_at = EXCLUDED.last_started_at,
			last_completed_at = EXCLUDED.last_completed_at,
			last_synced_at = EXCLUDED.last_synced_at,
			last_status = EXCLUDED.last_status,
			last_error = EXCLUDED.last_error,
			source_server_code = EXCLUDED.source_server_code,
			district_level_name = EXCLUDED.district_level_name,
			district_level_code = EXCLUDED.district_level_code,
			last_counts = EXCLUDED.last_counts,
			updated_at = NOW()
	`,
		input.StartedAt,
		input.CompletedAt,
		successTime(input),
		result.Status,
		result.ErrorMessage,
		result.ServerCode,
		result.DistrictLevelName,
		result.DistrictLevelCode,
		counts,
	)
	return err
}

func successTime(input replaceHierarchyInput) *time.Time {
	if input.Status != syncStatusSucceeded {
		return nil
	}
	return input.CompletedAt
}

func countRows(ctx context.Context, tx *sqlx.Tx, query string) (int, error) {
	var count int
	if err := tx.GetContext(ctx, &count, query); err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return count, nil
}

func groupMemberCount(items map[string][]string) int {
	total := 0
	for _, groupUIDs := range items {
		total += len(groupUIDs)
	}
	return total
}

func cloneJSONMap(input JSONMap) JSONMap {
	if input == nil {
		return JSONMap{}
	}
	cloned := JSONMap{}
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
