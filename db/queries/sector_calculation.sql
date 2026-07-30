-- name: InsertMarketCalculationWindow :exec
INSERT INTO market_calculation_windows (
    id, daily_tide_id, provider_key, data_window_start, data_window_end,
    required_granularity_seconds, observation_tolerance_seconds
) VALUES (
    sqlc.arg(id), sqlc.arg(daily_tide_id), sqlc.arg(provider_key), sqlc.arg(data_window_start), sqlc.arg(data_window_end),
    sqlc.arg(required_granularity_seconds), sqlc.arg(observation_tolerance_seconds)
);

-- name: InsertMarketPriceObservation :exec
INSERT INTO market_price_observations (
    id, calculation_window_id, market_asset_id, observed_at, raw_price,
    price, quality_status, quality_flags, raw_response_hash
) VALUES (
    sqlc.arg(id), sqlc.arg(calculation_window_id), sqlc.arg(market_asset_id), sqlc.arg(observed_at), sqlc.arg(raw_price),
    sqlc.narg(price), sqlc.arg(quality_status), sqlc.arg(quality_flags), sqlc.narg(raw_response_hash)
);

-- name: ListMarketPriceObservationsForAsset :many
SELECT id, calculation_window_id, market_asset_id, observed_at, raw_price, price,
       quality_status, quality_flags, raw_response_hash, created_at
FROM market_price_observations
WHERE calculation_window_id = sqlc.arg(calculation_window_id)
  AND market_asset_id = sqlc.arg(market_asset_id)
ORDER BY observed_at ASC, id ASC;

-- name: GetBasketCalculationMapping :one
SELECT
    mapping.id AS basket_mapping_version_id,
    mapping.mapping_key,
    mapping.content_version,
    mapping.minimum_covered_weight_units,
    mapping.normalization_cap_units,
    mapping.benchmark_eligible,
    definition.id AS sector_definition_version_id,
    definition.sector_key,
    definition.benchmark_method,
    definition.minimum_eligible_baskets,
    definition.relative_scale_units,
    definition.relative_blend_weight_units,
    definition.rank_blend_weight_units,
    definition.score_cap_units,
    policy.id AS expected_turbulence_policy_id,
    policy.policy_key AS expected_turbulence_policy_key,
    policy.policy_type AS expected_turbulence_policy_type,
    policy.static_value_units AS expected_turbulence_static_value_units,
    policy.floor_units AS expected_turbulence_floor_units,
    policy.rounding_mode AS expected_turbulence_rounding_mode
FROM basket_mapping_versions AS mapping
JOIN sector_definition_versions AS definition
    ON definition.id = mapping.sector_definition_version_id
JOIN expected_turbulence_policies AS policy
    ON policy.id = mapping.expected_turbulence_policy_id
WHERE mapping.id = sqlc.arg(basket_mapping_version_id);

-- name: ListBasketCalculationMappingComponents :many
SELECT component.basket_mapping_version_id, component.market_asset_id, component.weight,
       component.created_at
FROM basket_mapping_components AS component
WHERE component.basket_mapping_version_id = sqlc.arg(basket_mapping_version_id)
ORDER BY component.market_asset_id ASC;

-- name: ListTargetBasketCalculationMappingsForContent :many
SELECT mapping.id, mapping.mapping_key, mapping.content_version, mapping.sector_definition_version_id,
       mapping.minimum_covered_weight_units, mapping.expected_turbulence_policy_id,
       mapping.normalization_cap_units, mapping.benchmark_eligible
FROM basket_mapping_versions AS mapping
JOIN sector_definition_versions AS definition
    ON definition.id = mapping.sector_definition_version_id
WHERE mapping.content_version = sqlc.arg(content_version)
ORDER BY definition.sector_key ASC, mapping.id ASC;

-- name: GetBasketMetricByTideAndMapping :one
SELECT id, daily_tide_id, basket_mapping_version_id, calculation_window_id, status,
       covered_weight_units, raw_return_units, expected_turbulence_units,
       normalized_performance_units, input_checksum, calculated_at, created_at
FROM basket_metrics
WHERE daily_tide_id = sqlc.arg(daily_tide_id)
  AND basket_mapping_version_id = sqlc.arg(basket_mapping_version_id);

-- name: InsertBasketMetric :exec
INSERT INTO basket_metrics (
    id, daily_tide_id, basket_mapping_version_id, calculation_window_id, status,
    covered_weight_units, raw_return_units, expected_turbulence_units,
    normalized_performance_units, input_checksum
) VALUES (
    sqlc.arg(id), sqlc.arg(daily_tide_id), sqlc.arg(basket_mapping_version_id), sqlc.arg(calculation_window_id), sqlc.arg(status),
    sqlc.arg(covered_weight_units), sqlc.narg(raw_return_units), sqlc.narg(expected_turbulence_units),
    sqlc.narg(normalized_performance_units), sqlc.arg(input_checksum)
);

-- name: InsertBasketMetricComponent :exec
INSERT INTO basket_metric_components (
    id, basket_metric_id, market_asset_id, status, target_weight_units,
    effective_weight_units, open_observation_id, close_observation_id,
    component_return_units, contribution_units, quality_flags
) VALUES (
    sqlc.arg(id), sqlc.arg(basket_metric_id), sqlc.arg(market_asset_id), sqlc.arg(status), sqlc.arg(target_weight_units),
    sqlc.narg(effective_weight_units), sqlc.narg(open_observation_id), sqlc.narg(close_observation_id),
    sqlc.narg(component_return_units), sqlc.narg(contribution_units), sqlc.arg(quality_flags)
);

-- name: ListBasketMetricComponents :many
SELECT id, basket_metric_id, market_asset_id, status, target_weight_units,
       effective_weight_units, open_observation_id, close_observation_id,
       component_return_units, contribution_units, quality_flags, created_at
FROM basket_metric_components
WHERE basket_metric_id = sqlc.arg(basket_metric_id)
ORDER BY market_asset_id ASC;

-- name: ListReadyBasketMetricsForSectorBenchmark :many
SELECT metric.id, metric.daily_tide_id, metric.basket_mapping_version_id,
       metric.calculation_window_id, metric.status, metric.covered_weight_units,
       metric.raw_return_units, metric.expected_turbulence_units,
       metric.normalized_performance_units, metric.input_checksum,
       metric.calculated_at, metric.created_at
FROM basket_metrics AS metric
JOIN basket_mapping_versions AS mapping
    ON mapping.id = metric.basket_mapping_version_id
WHERE metric.daily_tide_id = sqlc.arg(daily_tide_id)
  AND mapping.sector_definition_version_id = sqlc.arg(sector_definition_version_id)
  AND mapping.benchmark_eligible = true
  AND metric.status = 'READY'
ORDER BY metric.basket_mapping_version_id ASC;

-- name: GetSectorBenchmarkByTideAndDefinition :one
SELECT id, daily_tide_id, sector_definition_version_id, status, calculation_method,
       eligible_basket_count, benchmark_units, minimum_required_baskets,
       input_checksum, calculated_at, created_at
FROM sector_benchmarks
WHERE daily_tide_id = sqlc.arg(daily_tide_id)
  AND sector_definition_version_id = sqlc.arg(sector_definition_version_id);

-- name: InsertSectorBenchmark :exec
INSERT INTO sector_benchmarks (
    id, daily_tide_id, sector_definition_version_id, status, calculation_method,
    eligible_basket_count, benchmark_units, minimum_required_baskets, input_checksum
) VALUES (
    sqlc.arg(id), sqlc.arg(daily_tide_id), sqlc.arg(sector_definition_version_id), sqlc.arg(status), sqlc.arg(calculation_method),
    sqlc.arg(eligible_basket_count), sqlc.narg(benchmark_units), sqlc.arg(minimum_required_baskets), sqlc.arg(input_checksum)
);

-- name: InsertSectorBenchmarkMember :exec
INSERT INTO sector_benchmark_members (
    id, sector_benchmark_id, basket_metric_id, benchmark_weight_units,
    normalized_performance_units, contribution_units, rank_index_units,
    rank_percentile_units
) VALUES (
    sqlc.arg(id), sqlc.arg(sector_benchmark_id), sqlc.arg(basket_metric_id), sqlc.arg(benchmark_weight_units),
    sqlc.arg(normalized_performance_units), sqlc.arg(contribution_units), sqlc.arg(rank_index_units),
    sqlc.arg(rank_percentile_units)
);

-- name: ListSectorBenchmarkMembers :many
SELECT id, sector_benchmark_id, basket_metric_id, benchmark_weight_units,
       normalized_performance_units, contribution_units, rank_index_units,
       rank_percentile_units, created_at
FROM sector_benchmark_members
WHERE sector_benchmark_id = sqlc.arg(sector_benchmark_id)
ORDER BY rank_index_units ASC, basket_metric_id ASC;

-- name: GetKeeperSectorResultByTideAndDefinition :one
SELECT id, daily_tide_id, keeper_definition_version_id, basket_metric_id,
       sector_benchmark_id, sector_relative_units, sector_percentile_units,
       relative_component_units, rank_component_units,
       sector_score_normalized_units, sector_score_points, created_at
FROM keeper_sector_results
WHERE daily_tide_id = sqlc.arg(daily_tide_id)
  AND keeper_definition_version_id = sqlc.arg(keeper_definition_version_id);

-- name: InsertKeeperSectorResult :exec
INSERT INTO keeper_sector_results (
    id, daily_tide_id, keeper_definition_version_id, basket_metric_id,
    sector_benchmark_id, sector_relative_units, sector_percentile_units,
    relative_component_units, rank_component_units,
    sector_score_normalized_units, sector_score_points
) VALUES (
    sqlc.arg(id), sqlc.arg(daily_tide_id), sqlc.arg(keeper_definition_version_id), sqlc.arg(basket_metric_id),
    sqlc.arg(sector_benchmark_id), sqlc.arg(sector_relative_units), sqlc.arg(sector_percentile_units),
    sqlc.arg(relative_component_units), sqlc.arg(rank_component_units),
    sqlc.arg(sector_score_normalized_units), sqlc.arg(sector_score_points)
);

-- name: ListTargetSectorBenchmarkReadiness :many
SELECT definition.sector_key, benchmark.status, benchmark.eligible_basket_count,
       definition.minimum_eligible_baskets AS minimum_required_baskets, benchmark.id AS sector_benchmark_id
FROM sector_definition_versions AS definition
LEFT JOIN sector_benchmarks AS benchmark
    ON benchmark.sector_definition_version_id = definition.id
   AND benchmark.daily_tide_id = sqlc.arg(daily_tide_id)
WHERE definition.content_version = sqlc.arg(content_version)
ORDER BY definition.sector_key ASC;
