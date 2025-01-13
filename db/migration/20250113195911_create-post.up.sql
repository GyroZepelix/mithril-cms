WITH inserted_post AS (
    INSERT INTO posts (id, title, slug, content, author_id, status, created_at, updated_at, published_at)
    VALUES 
        ('9ec7ab6d-e7a4-46f8-aae2-70aa7d4112e0', 'Is white rice good?', 'is-white-rice-good', 'Lorem ipsum', 'e45b9f41-c48a-4287-96ec-f15675ebcac3', 'published', '2024-08-18 18:02:35', '2024-08-18 18:02:35', '2024-08-18 18:02:35'),
        ('77e7a7bd-0b8d-4f45-8564-494e0c0096c9', 'Birds spotted in the park!', 'birds-spotted-in-the-park', 'Lorem ipsum', 'efad0618-1e42-4b41-8dc6-0bdf36eeaa1d', 'published', '2024-10-12 06:42:11', '2024-10-12 06:42:11', '2024-10-12 06:42:11'),
        ('8847ded1-887e-4a6f-af5b-65c7519c7628', 'How to create Bread', 'how-to-create-bread', 'Lorem ipsum', 'efad0618-1e42-4b41-8dc6-0bdf36eeaa1d', 'published', '2025-01-13 20:02:35', '2025-01-13 20:02:35', '2025-01-13 20:02:35')
    RETURNING id, author_id
)
UPDATE users u
SET posts = array_cat(u.posts, ARRAY(
    SELECT id FROM inserted_post WHERE author_id = u.id
))
WHERE id IN (SELECT DISTINCT author_id FROM inserted_post);

INSERT INTO categories (name, slug)
VALUES
    ('Food', 'food'),
    ('Review', 'review'),
    ('Tutorial', 'tutorial'),
    ('Nature', 'nature');

INSERT INTO post_categories (post_id, category_id)
VALUES
    ('9ec7ab6d-e7a4-46f8-aae2-70aa7d4112e0', 1), 
    ('9ec7ab6d-e7a4-46f8-aae2-70aa7d4112e0', 2), 
    ('77e7a7bd-0b8d-4f45-8564-494e0c0096c9', 4), 
    ('8847ded1-887e-4a6f-af5b-65c7519c7628', 1), 
    ('8847ded1-887e-4a6f-af5b-65c7519c7628', 3); 
