-- Script to reset releases processing steps to allow re-processing
-- This will mark the following steps as incomplete so they will be re-run:
-- - releases_processing
-- - release_genres_collection
-- - release_genres_upsert
-- - release_genre_associations
-- - release_label_associations
-- - release_artist_associations

-- Update the latest processing record to mark releases steps as incomplete
UPDATE discogs_data_processings
SET
    processing_stats = jsonb_set(
        jsonb_set(
            jsonb_set(
                jsonb_set(
                    jsonb_set(
                        jsonb_set(
                            processing_stats,
                            '{processing_steps,releases_processing}',
                            '{"completed": false}'::jsonb
                        ),
                        '{processing_steps,release_genres_collection}',
                        '{"completed": false}'::jsonb
                    ),
                    '{processing_steps,release_genres_upsert}',
                    '{"completed": false}'::jsonb
                ),
                '{processing_steps,release_genre_associations}',
                '{"completed": false}'::jsonb
            ),
            '{processing_steps,release_label_associations}',
            '{"completed": false}'::jsonb
        ),
        '{processing_steps,release_artist_associations}',
        '{"completed": false}'::jsonb
    ),
    updated_at = NOW()
WHERE id = (
    SELECT id
    FROM discogs_data_processings
    WHERE status IN ('completed', 'processing')
    ORDER BY created_at DESC
    LIMIT 1
)
RETURNING id, year_month, status;

-- Verify the update
SELECT
    id,
    year_month,
    status,
    processing_stats->'processing_steps'->'releases_processing' as releases_step,
    processing_stats->'processing_steps'->'release_artist_associations' as artist_assoc_step
FROM discogs_data_processings
WHERE id = (
    SELECT id
    FROM discogs_data_processings
    ORDER BY created_at DESC
    LIMIT 1
);
