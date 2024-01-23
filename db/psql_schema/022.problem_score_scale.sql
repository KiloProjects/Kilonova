
ALTER TABLE problems ADD COLUMN leaderboard_score_scale numeric NOT NULL DEFAULT 100;
ALTER TABLE submissions ADD COLUMN leaderboard_score_scale numeric NOT NULL DEFAULT 100;
ALTER TABLE submission_subtasks ADD COLUMN leaderboard_score_scale numeric NOT NULL DEFAULT 100;


