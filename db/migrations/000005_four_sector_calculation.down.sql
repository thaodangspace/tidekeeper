DROP TABLE IF EXISTS keeper_sector_results;
DROP TABLE IF EXISTS sector_benchmark_members;
DROP TABLE IF EXISTS sector_benchmarks;
DROP TABLE IF EXISTS basket_metric_components;
DROP TABLE IF EXISTS basket_metrics;
DROP TABLE IF EXISTS market_price_observations;
DROP TABLE IF EXISTS market_calculation_windows;

DROP TRIGGER IF EXISTS expected_turbulence_policies_draft_only_trigger ON expected_turbulence_policies;
DROP TRIGGER IF EXISTS sector_definition_versions_draft_only_trigger ON sector_definition_versions;

ALTER TABLE IF EXISTS basket_mapping_versions
    DROP CONSTRAINT IF EXISTS basket_mapping_versions_calculation_policy_shape_check,
    DROP CONSTRAINT IF EXISTS basket_mapping_versions_turbulence_policy_content_fk,
    DROP CONSTRAINT IF EXISTS basket_mapping_versions_sector_definition_content_fk;
DROP INDEX IF EXISTS basket_mapping_versions_sector_definition_idx;
ALTER TABLE IF EXISTS basket_mapping_versions
    DROP COLUMN IF EXISTS benchmark_eligible,
    DROP COLUMN IF EXISTS normalization_cap_units,
    DROP COLUMN IF EXISTS expected_turbulence_policy_id,
    DROP COLUMN IF EXISTS minimum_covered_weight_units,
    DROP COLUMN IF EXISTS sector_definition_version_id;

DROP TABLE IF EXISTS expected_turbulence_policies;
DROP TABLE IF EXISTS sector_definition_versions;
DROP TABLE IF EXISTS sectors;
