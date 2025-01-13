DELETE FROM post_categories
WHERE (post_id, category_id) IN
    ((1, 1), (1, 2), (2, 4), (3, 1), (3, 3));

DELETE FROM categories
WHERE slug IN ('food', 'review', 'tutorial', 'nature');

UPDATE users u
SET posts = array_remove(u.posts, p.id)
FROM posts p
WHERE p.slug IN (
    'is-white-rice-good',
    'birds-spotted-in-the-park',
    'how-to-create-bread'
) AND p.author_id = u.id;

DELETE FROM posts
WHERE slug IN (
    'is-white-rice-good',
    'birds-spotted-in-the-park',
    'how-to-create-bread'
);
