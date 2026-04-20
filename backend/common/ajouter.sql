CREATE TABLE public.groupe (
    id bigint NOT NULL,
    name character varying(255),
    promo_id bigint NOT NULL REFERENCES public.promotion(id) ON DELETE cascade,
    CONSTRAINT groupe_pkey PRIMARY KEY (id)
);

CREATE TABLE public.eleve_groupe (
	num_etudiant int4 NOT NULL,
	id_groupe int4 NOT NULL,
	CONSTRAINT pk_eleve_groupe PRIMARY KEY (num_etudiant, id_groupe),
	CONSTRAINT fk_eleve_groupe_eleve FOREIGN KEY (num_etudiant) REFERENCES public.user(id),
	CONSTRAINT fk_eleve_groupe_groupe FOREIGN KEY (id_groupe) REFERENCES public.groupe(id)
);


ALTER TABLE public.feedback DROP CONSTRAINT feedback_student_id_fkey;
ALTER TABLE public.feedback ADD CONSTRAINT feedback_student_id_fkey FOREIGN KEY (num_etudiant) REFERENCES public."user"(id) ON DELETE CASCADE;

ALTER TABLE public.user ADD COLUMN blame BOOLEAN DEFAULT FALSE;

ALTER TABLE public.student DROP CONSTRAINT elo_student_num_etudiant_fkey;
ALTER TABLE public.student ADD CONSTRAINT elo_student_num_etudiant_fkey FOREIGN KEY (num_etudiant) REFERENCES public."user"(id) ON DELETE CASCADE;

-- RGPD : stocker promotion et groupe dans feedback_classification pour pouvoir supprimer user_id
ALTER TABLE public.feedback_classification
    ADD COLUMN IF NOT EXISTS promotion CHARACTER VARYING(255),
    ADD COLUMN IF NOT EXISTS groupe CHARACTER VARYING(255);
