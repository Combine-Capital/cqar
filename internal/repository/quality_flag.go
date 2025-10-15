package repository

import (
	"context"
	"fmt"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RaiseQualityFlag creates a new quality flag for an asset
func (r *PostgresRepository) RaiseQualityFlag(ctx context.Context, flag *assetsv1.AssetQualityFlag) error {
	// Validate that the asset exists
	var assetExists bool
	err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", flag.GetAssetId()).Scan(&assetExists)
	if err != nil {
		return fmt.Errorf("check asset exists: %w", err)
	}
	if !assetExists {
		return fmt.Errorf("asset not found: %s", flag.GetAssetId())
	}

	// Generate ID if not provided
	if flag.FlagId == nil || *flag.FlagId == "" {
		id := uuid.New().String()
		flag.FlagId = &id
	}

	// Set raised_at timestamp if not provided
	if flag.RaisedAt == nil {
		flag.RaisedAt = timestamppb.Now()
	}

	query := `
		INSERT INTO quality_flags (
			id, asset_id, flag_type, severity, source, reason,
			evidence_url, raised_at, resolved_at, resolved_by, resolution_notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	// Convert enums to strings
	flagTypeStr := flag.GetFlagType().String()
	severityStr := flag.GetSeverity().String()

	_, err = r.exec(ctx, query,
		flag.GetFlagId(),
		flag.GetAssetId(),
		flagTypeStr,
		severityStr,
		flag.GetSource(),
		flag.GetReason(),
		nullableString(flag.EvidenceUrl),
		flag.RaisedAt.AsTime(),
		nullableTimestamp(flag.ResolvedAt),
		nullableString(flag.ResolvedBy),
		nullableString(flag.ResolutionNotes),
	)

	if err != nil {
		return fmt.Errorf("raise quality flag: %w", err)
	}

	return nil
}

// ResolveQualityFlag resolves an existing quality flag
func (r *PostgresRepository) ResolveQualityFlag(ctx context.Context, id string, resolvedBy string, resolutionNotes string) error {
	query := `
		UPDATE quality_flags
		SET
			resolved_at = $1,
			resolved_by = $2,
			resolution_notes = $3
		WHERE id = $4 AND resolved_at IS NULL
	`

	result, err := r.exec(ctx, query, time.Now(), resolvedBy, resolutionNotes, id)
	if err != nil {
		return fmt.Errorf("resolve quality flag: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("quality flag not found or already resolved: %s", id)
	}

	return nil
}

// GetQualityFlag retrieves a quality flag by ID
func (r *PostgresRepository) GetQualityFlag(ctx context.Context, id string) (*assetsv1.AssetQualityFlag, error) {
	query := `
		SELECT
			id, asset_id, flag_type, severity, source, reason,
			evidence_url, raised_at, resolved_at, resolved_by, resolution_notes
		FROM quality_flags
		WHERE id = $1
	`

	var flagId, assetId, flagTypeStr, severityStr, source, reason string
	var evidenceUrl, resolvedBy, resolutionNotes *string
	var raisedAt time.Time
	var resolvedAt *time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&flagId,
		&assetId,
		&flagTypeStr,
		&severityStr,
		&source,
		&reason,
		&evidenceUrl,
		&raisedAt,
		&resolvedAt,
		&resolvedBy,
		&resolutionNotes,
	)

	if err != nil {
		return nil, fmt.Errorf("get quality flag: %w", err)
	}

	// Parse enums
	flagType := parseFlagType(flagTypeStr)
	severity := parseFlagSeverity(severityStr)

	flag := &assetsv1.AssetQualityFlag{
		FlagId:          &flagId,
		AssetId:         &assetId,
		FlagType:        &flagType,
		Severity:        &severity,
		Source:          &source,
		Reason:          &reason,
		EvidenceUrl:     evidenceUrl,
		RaisedAt:        timestamppb.New(raisedAt),
		ResolvedAt:      timeToTimestampPtr(resolvedAt),
		ResolvedBy:      resolvedBy,
		ResolutionNotes: resolutionNotes,
	}

	return flag, nil
}

// ListQualityFlags retrieves a list of quality flags with optional filtering
func (r *PostgresRepository) ListQualityFlags(ctx context.Context, filter *QualityFlagFilter) ([]*assetsv1.AssetQualityFlag, error) {
	// Build query with filters
	query := `
		SELECT
			id, asset_id, flag_type, severity, source, reason,
			evidence_url, raised_at, resolved_at, resolved_by, resolution_notes
		FROM quality_flags
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.AssetID != nil {
			query += fmt.Sprintf(" AND asset_id = $%d", argPos)
			args = append(args, *filter.AssetID)
			argPos++
		}

		if filter.FlagType != nil {
			query += fmt.Sprintf(" AND flag_type = $%d", argPos)
			args = append(args, *filter.FlagType)
			argPos++
		}

		if filter.Severity != nil {
			query += fmt.Sprintf(" AND severity = $%d", argPos)
			args = append(args, *filter.Severity)
			argPos++
		}

		if filter.ActiveOnly {
			query += " AND resolved_at IS NULL"
		}

		// Add sorting
		query += " ORDER BY raised_at DESC"

		// Add pagination
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list quality flags: %w", err)
	}
	defer rows.Close()

	var flags []*assetsv1.AssetQualityFlag
	for rows.Next() {
		var flagId, assetId, flagTypeStr, severityStr, source, reason string
		var evidenceUrl, resolvedBy, resolutionNotes *string
		var raisedAt time.Time
		var resolvedAt *time.Time

		err := rows.Scan(
			&flagId,
			&assetId,
			&flagTypeStr,
			&severityStr,
			&source,
			&reason,
			&evidenceUrl,
			&raisedAt,
			&resolvedAt,
			&resolvedBy,
			&resolutionNotes,
		)
		if err != nil {
			return nil, fmt.Errorf("scan quality flag row: %w", err)
		}

		// Parse enums
		flagType := parseFlagType(flagTypeStr)
		severity := parseFlagSeverity(severityStr)

		flag := &assetsv1.AssetQualityFlag{
			FlagId:          &flagId,
			AssetId:         &assetId,
			FlagType:        &flagType,
			Severity:        &severity,
			Source:          &source,
			Reason:          &reason,
			EvidenceUrl:     evidenceUrl,
			RaisedAt:        timestamppb.New(raisedAt),
			ResolvedAt:      timeToTimestampPtr(resolvedAt),
			ResolvedBy:      resolvedBy,
			ResolutionNotes: resolutionNotes,
		}

		flags = append(flags, flag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quality flags: %w", err)
	}

	return flags, nil
}

// parseFlagType converts database string to FlagType enum
func parseFlagType(s string) assetsv1.FlagType {
	switch s {
	case "FLAG_TYPE_SCAM", "SCAM":
		return assetsv1.FlagType_FLAG_TYPE_SCAM
	case "FLAG_TYPE_RUGPULL", "RUGPULL":
		return assetsv1.FlagType_FLAG_TYPE_RUGPULL
	case "FLAG_TYPE_EXPLOITED", "EXPLOITED":
		return assetsv1.FlagType_FLAG_TYPE_EXPLOITED
	case "FLAG_TYPE_DEPRECATED", "DEPRECATED":
		return assetsv1.FlagType_FLAG_TYPE_DEPRECATED
	case "FLAG_TYPE_PAUSED", "PAUSED":
		return assetsv1.FlagType_FLAG_TYPE_PAUSED
	case "FLAG_TYPE_UNVERIFIED", "UNVERIFIED":
		return assetsv1.FlagType_FLAG_TYPE_UNVERIFIED
	case "FLAG_TYPE_LOW_LIQUIDITY", "LOW_LIQUIDITY":
		return assetsv1.FlagType_FLAG_TYPE_LOW_LIQUIDITY
	case "FLAG_TYPE_HONEYPOT", "HONEYPOT":
		return assetsv1.FlagType_FLAG_TYPE_HONEYPOT
	case "FLAG_TYPE_TAX_TOKEN", "TAX_TOKEN":
		return assetsv1.FlagType_FLAG_TYPE_TAX_TOKEN
	default:
		return assetsv1.FlagType_FLAG_TYPE_UNSPECIFIED
	}
}

// parseFlagSeverity converts database string to FlagSeverity enum
func parseFlagSeverity(s string) assetsv1.FlagSeverity {
	switch s {
	case "FLAG_SEVERITY_INFO", "INFO":
		return assetsv1.FlagSeverity_FLAG_SEVERITY_INFO
	case "FLAG_SEVERITY_LOW", "LOW":
		return assetsv1.FlagSeverity_FLAG_SEVERITY_LOW
	case "FLAG_SEVERITY_MEDIUM", "MEDIUM", "WARNING":
		return assetsv1.FlagSeverity_FLAG_SEVERITY_MEDIUM
	case "FLAG_SEVERITY_HIGH", "HIGH":
		return assetsv1.FlagSeverity_FLAG_SEVERITY_HIGH
	case "FLAG_SEVERITY_CRITICAL", "CRITICAL":
		return assetsv1.FlagSeverity_FLAG_SEVERITY_CRITICAL
	default:
		return assetsv1.FlagSeverity_FLAG_SEVERITY_UNSPECIFIED
	}
}

// Helper function to handle nullable timestamp values
func nullableTimestamp(ts *timestamppb.Timestamp) interface{} {
	if ts == nil {
		return nil
	}
	return ts.AsTime()
}

// Helper function to convert *time.Time to *timestamppb.Timestamp
func timeToTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
