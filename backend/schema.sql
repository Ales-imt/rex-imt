CREATE TABLE public.user (
    id serial4 NOT NULL,
	"version" int4 DEFAULT 1 NULL,
	"name" varchar(255) NOT NULL,
	surname varchar(255) NOT NULL,
	email varchar(255) NOT NULL,
	roles varchar(255) DEFAULT 'user'::character varying NULL,
	blame bool DEFAULT false NULL,
	CONSTRAINT user_email_key UNIQUE (email),
	CONSTRAINT user_pkey PRIMARY KEY (id)
);

CREATE TABLE public.student (
	user_id int4 UNIQUE NOT NULL,
	elo int4 DEFAULT 1000 NOT NULL,
	last_elo_gain int4 DEFAULT 0 NULL,
	promotion varchar(255) DEFAULT NULL,
	CONSTRAINT elo_student_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.user(id)
);

CREATE TABLE public.refresh_tokens (
    id serial4 NOT NULL,
    user_id serial4 NOT NULL REFERENCES public.user(id) ON DELETE CASCADE,
	token_version integer default '1',
	session text NOT NULL,
    token text UNIQUE NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);


CREATE TABLE public.season (
	id serial4 NOT NULL,
	start_date timestamp with time zone not null,
	end_date timestamp with time zone NOT NULL,
	CONSTRAINT season_pkey PRIMARY KEY (id)
);

-- public.elo_history definition

-- Drop table

-- DROP TABLE public.elo_history;

CREATE TABLE public.elo_history (
	id serial4 NOT NULL,
	user_id int4 NULL,
	season_id int4 NULL,
	final_elo int4 NOT NULL,
	rank int4 NULL,
	CONSTRAINT elo_history_pkey PRIMARY KEY (id),
	CONSTRAINT elo_history_season_id_fkey FOREIGN KEY (season_id) REFERENCES public.season(id),
	CONSTRAINT elo_history_student_id_fkey FOREIGN KEY (user_id) REFERENCES public.user(id)
);


-- public.feedback definition

-- Drop table

-- DROP TABLE public.feedback;

CREATE TABLE public.feedback (
	id serial4 NOT NULL,
	user_id int4 NULL,
	content text NOT NULL,
	created_at timestamp with time zone DEFAULT now() NULL,
	season_id int4 NULL,
	CONSTRAINT feedback_pkey PRIMARY KEY (id),
	CONSTRAINT feedback_season_id_fkey FOREIGN KEY (season_id) REFERENCES public.season(id),
	CONSTRAINT feedback_student_id_fkey FOREIGN KEY (user_id) REFERENCES public.user(id)
);


CREATE TABLE public.promotion (
    id bigint NOT NULL,
    name character varying(255),
    CONSTRAINT promotion_pkey PRIMARY KEY (id)
);

CREATE TABLE public.groupe (
    id bigint NOT NULL,
    name character varying(255),
    promo_id bigint NOT NULL REFERENCES public.promotion(id) ON DELETE cascade,
    CONSTRAINT groupe_pkey PRIMARY KEY (id)
);

CREATE TABLE feedback_groupe (
    feedback_id   INT NOT NULL REFERENCES public.feedback(id) ON DELETE CASCADE,
    groupe_id INT NOT NULL REFERENCES groupe(id) ON DELETE CASCADE,
    PRIMARY KEY (feedback_id, groupe_id)
);

CREATE TABLE public.eleve_groupe (
	num_etudiant int4 NOT NULL,
	id_groupe int4 NOT NULL,
	CONSTRAINT pk_eleve_groupe PRIMARY KEY (num_etudiant, id_groupe),
	CONSTRAINT fk_eleve_groupe_eleve FOREIGN KEY (num_etudiant) REFERENCES public.user(id),
	CONSTRAINT fk_eleve_groupe_groupe FOREIGN KEY (id_groupe) REFERENCES public.groupe(id)
);

