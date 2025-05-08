-- Remove entries from post_categories first to avoid foreign key constraint violations
DELETE FROM post_categories WHERE post_id IN (
    '9ec7ab6d-e7a4-46f8-aae2-70aa7d4112e0',
    '77e7a7bd-0b8d-4f45-8564-494e0c0096c9',
    '8847ded1-887e-4a6f-af5b-65c7519c7628'
);

-- Remove posts from users' posts array
UPDATE users u
SET posts = array_remove(posts, id)
WHERE id IN (
    'e45b9f41-c48a-4287-96ec-f15675ebcac3',
    'efad0618-1e42-4b41-8dc6-0bdf36eeaa1d'
)
AND id IN (
    '9ec7ab6d-e7a4-46f8-aae2-70aa7d4112e0',
    '77e7a7bd-0b8d-4f45-8564-494e0c0096c9',
    '8847ded1-887e-4a6f-af5b-65c7519c7628'
);

-- Delete the posts
DELETE FROM posts WHERE id IN (
    '9ec7ab6d-e7a4-46f8-aae2-70aa7d4112e0',
    '77e7a7bd-0b8d-4f45-8564-494e0c0096c9',
    '8847ded1-887e-4a6f-af5b-65c7519c7628'
);

-- Remove categories if they were added only in this migration (optional: check dependencies before deletion)
DELETE FROM categories WHERE slug IN ('food', 'review', 'tutorial', 'nature');
