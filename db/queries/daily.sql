-- name: GetCurrentDailyContextForPlayer :one
SELECT
    transaction_timestamp()::timestamptz AS server_now,
    p.current_voyage_id,
    v.public_id AS voyage_public_id,
    v.status AS voyage_status,
    v.current_day_number,
    v.fund_health,
    v.max_fund_health,
    v.capital,
    COALESCE(v.score::text, '') AS score,
    v.row_version AS voyage_row_version,
    s.id AS player_daily_state_id,
    s.phase AS player_phase,
    s.version AS player_state_version,
    s.selected_strategy_id,
    s.pending_reward_count,
    t.id AS daily_tide_id,
    t.day_key AS tide_day_key,
    t.sequence_number AS tide_sequence_number,
    t.phase AS tide_phase,
    t.lock_at,
    t.settle_after,
    t.content_version AS tide_content_version,
    cv.schema_version AS projection_schema_version,
    cv.projection,
    cv.validated_at AS projection_validated_at
FROM players AS p
LEFT JOIN voyages AS v
    ON v.id = p.current_voyage_id
   AND v.player_id = p.id
LEFT JOIN player_daily_states AS s
    ON s.player_id = p.id
   AND s.voyage_id = v.id
   AND s.day_number = v.current_day_number
LEFT JOIN daily_tides AS t
    ON t.id = s.daily_tide_id
LEFT JOIN player_daily_context_views AS cv
    ON cv.player_daily_state_id = s.id
WHERE p.id = sqlc.arg(player_id);
