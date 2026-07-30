-- Four-sector calculation foundation. All configuration is additive and tied to
-- immutable content releases; published v1 catalog rows remain unconfigured.

CREATE TABLE sectors (
    sector_key text PRIMARY KEY,
    name_key text NOT NULL,
    summary_key text NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE',
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT sectors_key_check
        CHECK (sector_key IN ('CREST', 'EMBER', 'CURRENT', 'HARBOR')),
    CONSTRAINT sectors_name_key_check
        CHECK (name_key ~ '^[a-z][a-z0-9_.]*$'),
    CONSTRAINT sectors_summary_key_check
        CHECK (summary_key ~ '^[a-z][a-z0-9_.]*$'),
    CONSTRAINT sectors_status_check
        CHECK (status IN ('ACTIVE', 'RETIRED')),
    CONSTRAINT sectors_timestamp_order_check
        CHECK (updated_at >= created_at)
);

INSERT INTO sectors (sector_key, name_key, summary_key)
VALUES
    ('CREST', 'sector.crest.name', 'sector.crest.summary'),
    ('EMBER', 'sector.ember.name', 'sector.ember.summary'),
    ('CURRENT', 'sector.current.name', 'sector.current.summary'),
    ('HARBOR', 'sector.harbor.name', 'sector.harbor.summary');

CREATE TABLE sector_definition_versions (
    id uuid PRIMARY KEY,
    sector_key text NOT NULL,
    content_version bigint NOT NULL,
    benchmark_method text NOT NULL,
    minimum_eligible_baskets integer NOT NULL,
    relative_scale_units bigint NOT NULL,
    relative_blend_weight_units bigint NOT NULL,
    rank_blend_weight_units bigint NOT NULL,
    score_cap_units bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT sector_definition_versions_sector_fk
        FOREIGN KEY (sector_key) REFERENCES sectors (sector_key)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sector_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sector_definition_versions_sector_content_key
        UNIQUE (sector_key, content_version),
    CONSTRAINT sector_definition_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT sector_definition_versions_benchmark_method_check
        CHECK (benchmark_method = 'EQUAL_WEIGHT'),
    CONSTRAINT sector_definition_versions_minimum_baskets_check
        CHECK (minimum_eligible_baskets >= 2),
    CONSTRAINT sector_definition_versions_relative_scale_check
        CHECK (relative_scale_units > 0),
    CONSTRAINT sector_definition_versions_blend_weights_check
        CHECK (relative_blend_weight_units >= 0
            AND rank_blend_weight_units >= 0
            AND relative_blend_weight_units + rank_blend_weight_units = 1000000),
    CONSTRAINT sector_definition_versions_score_cap_check
        CHECK (score_cap_units > 0)
);

CREATE INDEX sector_definition_versions_content_idx
    ON sector_definition_versions (content_version, sector_key);

CREATE TABLE expected_turbulence_policies (
    id uuid PRIMARY KEY,
    policy_key text NOT NULL,
    content_version bigint NOT NULL,
    policy_type text NOT NULL,
    static_value_units bigint,
    floor_units bigint NOT NULL,
    rounding_mode text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT expected_turbulence_policies_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT expected_turbulence_policies_key_content_key
        UNIQUE (policy_key, content_version),
    CONSTRAINT expected_turbulence_policies_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT expected_turbulence_policies_key_check
        CHECK (policy_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT expected_turbulence_policies_type_check
        CHECK (policy_type = 'STATIC_CONTENT_VALUE'),
    CONSTRAINT expected_turbulence_policies_static_value_check
        CHECK (static_value_units IS NOT NULL AND static_value_units > 0),
    CONSTRAINT expected_turbulence_policies_floor_check
        CHECK (floor_units > 0 AND floor_units <= static_value_units),
    CONSTRAINT expected_turbulence_policies_rounding_check
        CHECK (rounding_mode = 'ROUND_HALF_AWAY_FROM_ZERO')
);

CREATE INDEX expected_turbulence_policies_content_idx
    ON expected_turbulence_policies (content_version, policy_key);

ALTER TABLE basket_mapping_versions
    ADD COLUMN sector_definition_version_id uuid,
    ADD COLUMN minimum_covered_weight_units bigint,
    ADD COLUMN expected_turbulence_policy_id uuid,
    ADD COLUMN normalization_cap_units bigint,
    ADD COLUMN benchmark_eligible boolean;

ALTER TABLE basket_mapping_versions
    ADD CONSTRAINT basket_mapping_versions_sector_definition_content_fk
        FOREIGN KEY (sector_definition_version_id, content_version)
        REFERENCES sector_definition_versions (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT basket_mapping_versions_turbulence_policy_content_fk
        FOREIGN KEY (expected_turbulence_policy_id, content_version)
        REFERENCES expected_turbulence_policies (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT basket_mapping_versions_calculation_policy_shape_check
        CHECK (
            (sector_definition_version_id IS NULL
                AND minimum_covered_weight_units IS NULL
                AND expected_turbulence_policy_id IS NULL
                AND normalization_cap_units IS NULL
                AND benchmark_eligible IS NULL)
            OR
            (sector_definition_version_id IS NOT NULL
                AND minimum_covered_weight_units BETWEEN 1 AND 100000000
                AND expected_turbulence_policy_id IS NOT NULL
                AND normalization_cap_units > 0
                AND benchmark_eligible IS NOT NULL)
        );

CREATE INDEX basket_mapping_versions_sector_definition_idx
    ON basket_mapping_versions (sector_definition_version_id)
    WHERE sector_definition_version_id IS NOT NULL;

CREATE TRIGGER sector_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON sector_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER expected_turbulence_policies_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON expected_turbulence_policies
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TABLE market_calculation_windows (
    id uuid PRIMARY KEY,
    daily_tide_id uuid NOT NULL,
    provider_key text NOT NULL,
    data_window_start timestamptz NOT NULL,
    data_window_end timestamptz NOT NULL,
    required_granularity_seconds integer NOT NULL,
    observation_tolerance_seconds integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT market_calculation_windows_tide_fk
        FOREIGN KEY (daily_tide_id) REFERENCES daily_tides (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT market_calculation_windows_tide_key UNIQUE (daily_tide_id),
    CONSTRAINT market_calculation_windows_provider_key_check
        CHECK (provider_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT market_calculation_windows_time_check
        CHECK (data_window_start < data_window_end),
    CONSTRAINT market_calculation_windows_granularity_check
        CHECK (required_granularity_seconds > 0),
    CONSTRAINT market_calculation_windows_tolerance_check
        CHECK (observation_tolerance_seconds >= 0)
);

CREATE TABLE market_price_observations (
    id uuid PRIMARY KEY,
    calculation_window_id uuid NOT NULL,
    market_asset_id uuid NOT NULL,
    observed_at timestamptz NOT NULL,
    raw_price text NOT NULL,
    price numeric(36,18),
    quality_status text NOT NULL,
    quality_flags jsonb NOT NULL DEFAULT '[]'::jsonb,
    raw_response_hash bytea,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT market_price_observations_window_fk
        FOREIGN KEY (calculation_window_id) REFERENCES market_calculation_windows (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT market_price_observations_asset_fk
        FOREIGN KEY (market_asset_id) REFERENCES market_assets (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT market_price_observations_window_asset_time_key
        UNIQUE (calculation_window_id, market_asset_id, observed_at),
    CONSTRAINT market_price_observations_raw_price_check
        CHECK (char_length(raw_price) BETWEEN 1 AND 256),
    CONSTRAINT market_price_observations_price_check
        CHECK (price IS NULL OR price > 0),
    CONSTRAINT market_price_observations_quality_status_check
        CHECK (quality_status IN ('VALID', 'INVALID_PRICE', 'OUTLIER_FLAGGED', 'PROVIDER_REJECTED')),
    CONSTRAINT market_price_observations_quality_flags_check
        CHECK (jsonb_typeof(quality_flags) = 'array'),
    CONSTRAINT market_price_observations_response_hash_check
        CHECK (raw_response_hash IS NULL OR octet_length(raw_response_hash) = 32)
);

CREATE INDEX market_price_observations_window_asset_time_idx
    ON market_price_observations (calculation_window_id, market_asset_id, observed_at);

CREATE TABLE basket_metrics (
    id uuid PRIMARY KEY,
    daily_tide_id uuid NOT NULL,
    basket_mapping_version_id uuid NOT NULL,
    calculation_window_id uuid NOT NULL,
    status text NOT NULL,
    covered_weight_units bigint NOT NULL,
    raw_return_units bigint,
    expected_turbulence_units bigint,
    normalized_performance_units bigint,
    input_checksum bytea NOT NULL,
    calculated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT basket_metrics_tide_fk
        FOREIGN KEY (daily_tide_id) REFERENCES daily_tides (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metrics_mapping_fk
        FOREIGN KEY (basket_mapping_version_id) REFERENCES basket_mapping_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metrics_window_fk
        FOREIGN KEY (calculation_window_id) REFERENCES market_calculation_windows (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metrics_tide_mapping_key
        UNIQUE (daily_tide_id, basket_mapping_version_id),
    CONSTRAINT basket_metrics_status_check
        CHECK (status IN ('READY', 'DATA_INCOMPLETE')),
    CONSTRAINT basket_metrics_covered_weight_check
        CHECK (covered_weight_units BETWEEN 0 AND 100000000),
    CONSTRAINT basket_metrics_ready_values_check
        CHECK (status <> 'READY'
            OR (raw_return_units IS NOT NULL
                AND expected_turbulence_units IS NOT NULL
                AND expected_turbulence_units > 0
                AND normalized_performance_units IS NOT NULL)),
    CONSTRAINT basket_metrics_checksum_check
        CHECK (octet_length(input_checksum) = 32)
);

CREATE INDEX basket_metrics_tide_status_idx
    ON basket_metrics (daily_tide_id, status, basket_mapping_version_id);

CREATE TABLE basket_metric_components (
    id uuid PRIMARY KEY,
    basket_metric_id uuid NOT NULL,
    market_asset_id uuid NOT NULL,
    status text NOT NULL,
    target_weight_units bigint NOT NULL,
    effective_weight_units bigint,
    open_observation_id uuid,
    close_observation_id uuid,
    component_return_units bigint,
    contribution_units bigint,
    quality_flags jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT basket_metric_components_metric_fk
        FOREIGN KEY (basket_metric_id) REFERENCES basket_metrics (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metric_components_asset_fk
        FOREIGN KEY (market_asset_id) REFERENCES market_assets (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metric_components_open_observation_fk
        FOREIGN KEY (open_observation_id) REFERENCES market_price_observations (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metric_components_close_observation_fk
        FOREIGN KEY (close_observation_id) REFERENCES market_price_observations (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_metric_components_metric_asset_key
        UNIQUE (basket_metric_id, market_asset_id),
    CONSTRAINT basket_metric_components_status_check
        CHECK (status IN ('VALID', 'MISSING_OPEN', 'MISSING_CLOSE', 'OUTSIDE_TOLERANCE',
            'INVALID_PRICE', 'OUTLIER_FLAGGED', 'PROVIDER_REJECTED', 'DISABLED_BY_MAPPING')),
    CONSTRAINT basket_metric_components_target_weight_check
        CHECK (target_weight_units BETWEEN 1 AND 100000000),
    CONSTRAINT basket_metric_components_effective_weight_check
        CHECK (effective_weight_units IS NULL
            OR effective_weight_units BETWEEN 1 AND 100000000),
    CONSTRAINT basket_metric_components_quality_flags_check
        CHECK (jsonb_typeof(quality_flags) = 'array')
);

CREATE INDEX basket_metric_components_metric_idx
    ON basket_metric_components (basket_metric_id);

CREATE TABLE sector_benchmarks (
    id uuid PRIMARY KEY,
    daily_tide_id uuid NOT NULL,
    sector_definition_version_id uuid NOT NULL,
    status text NOT NULL,
    calculation_method text NOT NULL,
    eligible_basket_count integer NOT NULL,
    benchmark_units bigint,
    minimum_required_baskets integer NOT NULL,
    input_checksum bytea NOT NULL,
    calculated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT sector_benchmarks_tide_fk
        FOREIGN KEY (daily_tide_id) REFERENCES daily_tides (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sector_benchmarks_definition_fk
        FOREIGN KEY (sector_definition_version_id) REFERENCES sector_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sector_benchmarks_tide_definition_key
        UNIQUE (daily_tide_id, sector_definition_version_id),
    CONSTRAINT sector_benchmarks_status_check
        CHECK (status IN ('READY', 'DATA_INCOMPLETE')),
    CONSTRAINT sector_benchmarks_calculation_method_check
        CHECK (calculation_method = 'EQUAL_WEIGHT'),
    CONSTRAINT sector_benchmarks_eligible_count_check
        CHECK (eligible_basket_count >= 0),
    CONSTRAINT sector_benchmarks_minimum_check
        CHECK (minimum_required_baskets >= 2),
    CONSTRAINT sector_benchmarks_ready_values_check
        CHECK (status <> 'READY'
            OR (eligible_basket_count >= minimum_required_baskets AND benchmark_units IS NOT NULL)),
    CONSTRAINT sector_benchmarks_checksum_check
        CHECK (octet_length(input_checksum) = 32)
);

CREATE INDEX sector_benchmarks_tide_status_idx
    ON sector_benchmarks (daily_tide_id, status, sector_definition_version_id);

CREATE TABLE sector_benchmark_members (
    id uuid PRIMARY KEY,
    sector_benchmark_id uuid NOT NULL,
    basket_metric_id uuid NOT NULL,
    benchmark_weight_units bigint NOT NULL,
    normalized_performance_units bigint NOT NULL,
    contribution_units bigint NOT NULL,
    rank_index_units bigint NOT NULL,
    rank_percentile_units bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT sector_benchmark_members_benchmark_fk
        FOREIGN KEY (sector_benchmark_id) REFERENCES sector_benchmarks (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sector_benchmark_members_metric_fk
        FOREIGN KEY (basket_metric_id) REFERENCES basket_metrics (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sector_benchmark_members_benchmark_metric_key
        UNIQUE (sector_benchmark_id, basket_metric_id),
    CONSTRAINT sector_benchmark_members_weight_check
        CHECK (benchmark_weight_units BETWEEN 1 AND 100000000),
    CONSTRAINT sector_benchmark_members_rank_index_check
        CHECK (rank_index_units >= 0),
    CONSTRAINT sector_benchmark_members_percentile_check
        CHECK (rank_percentile_units BETWEEN 0 AND 1000000)
);

CREATE INDEX sector_benchmark_members_benchmark_rank_idx
    ON sector_benchmark_members (sector_benchmark_id, rank_index_units, basket_metric_id);

CREATE TABLE keeper_sector_results (
    id uuid PRIMARY KEY,
    daily_tide_id uuid NOT NULL,
    keeper_definition_version_id uuid NOT NULL,
    basket_metric_id uuid NOT NULL,
    sector_benchmark_id uuid NOT NULL,
    sector_relative_units bigint NOT NULL,
    sector_percentile_units bigint NOT NULL,
    relative_component_units bigint NOT NULL,
    rank_component_units bigint NOT NULL,
    sector_score_normalized_units bigint NOT NULL,
    sector_score_points bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT keeper_sector_results_tide_fk
        FOREIGN KEY (daily_tide_id) REFERENCES daily_tides (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_sector_results_definition_fk
        FOREIGN KEY (keeper_definition_version_id) REFERENCES keeper_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_sector_results_metric_fk
        FOREIGN KEY (basket_metric_id) REFERENCES basket_metrics (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_sector_results_benchmark_fk
        FOREIGN KEY (sector_benchmark_id) REFERENCES sector_benchmarks (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_sector_results_tide_definition_key
        UNIQUE (daily_tide_id, keeper_definition_version_id),
    CONSTRAINT keeper_sector_results_percentile_check
        CHECK (sector_percentile_units BETWEEN 0 AND 1000000),
    CONSTRAINT keeper_sector_results_relative_component_check
        CHECK (relative_component_units BETWEEN -1000000 AND 1000000),
    CONSTRAINT keeper_sector_results_rank_component_check
        CHECK (rank_component_units BETWEEN -1000000 AND 1000000),
    CONSTRAINT keeper_sector_results_normalized_score_check
        CHECK (sector_score_normalized_units BETWEEN -1000000 AND 1000000)
);

CREATE INDEX keeper_sector_results_tide_benchmark_idx
    ON keeper_sector_results (daily_tide_id, sector_benchmark_id);
