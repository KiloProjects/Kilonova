
CREATE TABLE IF NOT EXISTS contests (
	id 					bigserial 	PRIMARY KEY,
	created_at			timestamptz	NOT NULL DEFAULT NOW(),
	name 				text 	  	NOT NULL UNIQUE,

    public_join         boolean     NOT NULL DEFAULT true,
    hidden              boolean     NOT NULL DEFAULT true,
    start_time          timestamptz NOT NULL,
    end_time            timestamptz NOT NULL,
    max_sub_count       integer     NOT NULL DEFAULT 30,
    CHECK (start_time <= end_time)
);

CREATE TABLE IF NOT EXISTS contest_user_access (
    contest_id  bigint           NOT NULL REFERENCES contests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id     bigint           NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    access      pbaccess_type    NOT NULL,

    UNIQUE (contest_id, user_id)
);

CREATE TABLE IF NOT EXISTS contest_problems (
    contest_id bigint NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    problem_id bigint NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    position bigint NOT NULL DEFAULT 0,
    UNIQUE (contest_id, problem_id)
);

CREATE TABLE IF NOT EXISTS contest_registrations (
	created_at	    timestamptz  NOT NULL DEFAULT NOW(),
    user_id         bigint       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id      bigint       NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    UNIQUE (user_id, contest_id)
);

CREATE TABLE IF NOT EXISTS contest_questions (
    author_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question       text   NOT NULL,
	created_at 	   timestamptz NOT NULL DEFAULT NOW(),

    response_author_id bigint REFERENCES users(id) ON DELETE SET NULL, 
    responded_at   timestamptz,
    response       text
);

CREATE TABLE IF NOT EXISTS contest_announcements (
    announcer_id   bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    announcement   text   NOT NULL,
	created_at 	   timestamptz NOT NULL DEFAULT NOW()
);

ALTER TABLE submissions ADD COLUMN contest_id bigint REFERENCES contests(id) ON DELETE SET NULL;
