
-- Note: This must be in sync with sudoapi/filters.go:IsBlogPostVisible
CREATE OR REPLACE FUNCTION visible_posts(user_id bigint) RETURNS TABLE (post_id bigint, user_id bigint) AS $$
    SELECT id AS post_id, $1 AS user_id FROM blog_posts 
        WHERE 
            visible = true OR 
            EXISTS (SELECT 1 FROM users WHERE id = $1 AND admin = true) OR 
            author_id = $1
$$ LANGUAGE SQL STABLE;
