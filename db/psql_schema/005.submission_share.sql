CREATE TABLE IF NOT EXISTS submission_pastes (
    paste_id        text    NOT NULL UNIQUE,
    submission_id   bigint  NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    author_id       bigint  NOT NULL REFERENCES users(id) ON DELETE CASCADE 
);