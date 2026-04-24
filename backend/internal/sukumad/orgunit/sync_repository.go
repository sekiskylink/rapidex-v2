package orgunit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

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
		if err := r.upsertSyncState(ctx, nil, input, result, 0, 0); err != nil {
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

	deletedAssignments, err := countRows(ctx, tx, `SELECT COUNT(*) FROM user_org_units`)
	if err != nil {
		return SyncResult{}, err
	}
	deletedReporters, err := countRows(ctx, tx, `SELECT COUNT(*) FROM reporters`)
	if err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_org_units`); err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM reporters`); err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM org_unit_group_members`); err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM org_unit_attributes`); err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM org_unit_groups`); err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM org_unit_levels`); err != nil {
		return SyncResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM org_units`); err != nil {
		return SyncResult{}, err
	}

	groupIDs := make(map[string]int64, len(input.Groups))
	for _, item := range input.Groups {
		var id int64
		if err := tx.QueryRowxContext(ctx, `
			INSERT INTO org_unit_groups (uid, code, name, short_name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id
		`, item.UID, item.Code, item.Name, item.ShortName).Scan(&id); err != nil {
			return SyncResult{}, err
		}
		groupIDs[item.UID] = id
	}
	for _, item := range input.Levels {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO org_unit_levels (uid, code, name, level, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, item.UID, item.Code, item.Name, item.Level); err != nil {
			return SyncResult{}, err
		}
	}
	for _, item := range input.Attributes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO org_unit_attributes (uid, code, name, short_name, value_type, is_unique, mandatory, organisation_unit_attribute, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		`, item.UID, item.Code, item.Name, item.ShortName, item.ValueType, item.IsUnique, item.Mandatory, item.OrganisationUnitAttribute); err != nil {
			return SyncResult{}, err
		}
	}

	orgUnitIDs := make(map[string]int64, len(input.OrgUnits))
	syncTime := input.StartedAt
	if input.CompletedAt != nil {
		syncTime = input.CompletedAt.UTC()
	}
	for _, item := range input.OrgUnits {
		var parentID *int64
		if parentUID, _ := item.Extras["parentUid"].(string); strings.TrimSpace(parentUID) != "" {
			id, ok := orgUnitIDs[parentUID]
			if !ok {
				return SyncResult{}, fmt.Errorf("parent org unit %s not inserted before child %s", parentUID, item.UID)
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
			return SyncResult{}, err
		}
		orgUnitIDs[item.UID] = id
	}
	for orgUnitUID, groupUIDs := range input.GroupMembers {
		orgUnitID, ok := orgUnitIDs[orgUnitUID]
		if !ok {
			return SyncResult{}, fmt.Errorf("group membership references unknown org unit %s", orgUnitUID)
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
				return SyncResult{}, err
			}
		}
	}

	result.DeletedAssignments = deletedAssignments
	result.DeletedReporters = deletedReporters
	if err := r.upsertSyncState(ctx, tx, input, result, deletedAssignments, deletedReporters); err != nil {
		return SyncResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return SyncResult{}, err
	}
	tx = nil
	return result, nil
}

func (r *PgRepository) upsertSyncState(ctx context.Context, tx *sqlx.Tx, input replaceHierarchyInput, result SyncResult, deletedAssignments int, deletedReporters int) error {
	counts := JSONMap{
		"levelsCount":        result.LevelsCount,
		"groupsCount":        result.GroupsCount,
		"attributesCount":    result.AttributesCount,
		"orgUnitsCount":      result.OrgUnitsCount,
		"groupMembersCount":  result.GroupMembersCount,
		"deletedAssignments": deletedAssignments,
		"deletedReporters":   deletedReporters,
		"dryRun":             result.DryRun,
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
	cloned := make(JSONMap, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
