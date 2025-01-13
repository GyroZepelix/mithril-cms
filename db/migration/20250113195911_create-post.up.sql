WITH inserted_post AS (
    INSERT INTO posts (title, slug, content, author_id, status, created_at, updated_at, published_at)
    VALUES 
        ('Is white rice good?', 'is-white-rice-good', 'Lorem ipsum', 1, 'published', '2024-08-18 18:02:35', '2024-08-18 18:02:35', '2024-08-18 18:02:35'),
        ('Birds spotted in the park!', 'birds-spotted-in-the-park', 'Lorem ipsum', 2, 'published', '2024-10-12 06:42:11', '2024-10-12 06:42:11', '2024-10-12 06:42:11'),
        ('How to create Bread', 'how-to-create-bread', 'Lorem ipsum', 2, 'published', '2025-01-13 20:02:35', '2025-01-13 20:02:35', '2025-01-13 20:02:35')
    RETURNING id, author_id
)
UPDATE users
SET posts = array_append(posts, inserted_post.id)
FROM inserted_post
WHERE users.id = inserted_post.author_id;

INSERT INTO categories (name, slug)
VALUES
    ('Food', 'food'),
    ('Review', 'review'),
    ('Tutorial', 'tutorial'),
    ('Nature', 'nature');

INSERT INTO post_categories (post_id, category_id)
VALUES
    (1, 1), 
    (1, 2), 
    (2, 4), 
    (3, 1), 
    (3, 3); 
