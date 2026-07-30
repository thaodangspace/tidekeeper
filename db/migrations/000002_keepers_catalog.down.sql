DROP TRIGGER IF EXISTS content_releases_published_immutable_trigger ON content_releases;
DROP TRIGGER IF EXISTS market_assets_immutable_trigger ON market_assets;
DROP TRIGGER IF EXISTS basket_mapping_components_weight_total_trigger ON basket_mapping_components;
DROP TRIGGER IF EXISTS basket_mapping_components_draft_only_trigger ON basket_mapping_components;
DROP TRIGGER IF EXISTS keeper_definition_versions_draft_only_trigger ON keeper_definition_versions;
DROP TRIGGER IF EXISTS basket_mapping_versions_draft_only_trigger ON basket_mapping_versions;

DROP FUNCTION IF EXISTS keeper_catalog_guard_published_release_mutation();
DROP FUNCTION IF EXISTS keeper_catalog_guard_asset_mutation();
DROP FUNCTION IF EXISTS keeper_catalog_assert_basket_weight_total();
DROP FUNCTION IF EXISTS keeper_catalog_guard_component_mutation();
DROP FUNCTION IF EXISTS keeper_catalog_component_content_version(uuid);
DROP FUNCTION IF EXISTS keeper_catalog_guard_release_mutation();
DROP FUNCTION IF EXISTS keeper_catalog_require_draft_content_release(bigint);

DROP TABLE IF EXISTS keeper_definition_versions;
DROP TABLE IF EXISTS basket_mapping_components;
DROP TABLE IF EXISTS basket_mapping_versions;
DROP TABLE IF EXISTS market_assets;
DROP TABLE IF EXISTS content_releases;
