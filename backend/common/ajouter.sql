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

CREATE TABLE feedback_groupe (
    feedback_id   INT NOT NULL REFERENCES public.feedback(id) ON DELETE CASCADE,
    groupe_id INT NOT NULL REFERENCES groupe(id) ON DELETE CASCADE,
    PRIMARY KEY (feedback_id, groupe_id)
);

-- Fonction qui copie les groupes de l'utilisateur dans feedback_groupe
CREATE OR REPLACE FUNCTION copy_eleve_groupes_to_feedback()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO feedback_groupe (feedback_id, groupe_id)
    SELECT NEW.id, eg.id_groupe
    FROM eleve_groupe eg
    WHERE eg.num_etudiant = NEW.user_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger sur l'insertion dans feedback
CREATE TRIGGER feedback_insert_copy_groupes
AFTER INSERT ON public.feedback
FOR EACH ROW
EXECUTE FUNCTION copy_eleve_groupes_to_feedback();

-- pour initialiser les feedbacks existants

INSERT INTO feedback_groupe (feedback_id, groupe_id)
SELECT f.id, eg.groupe_id
FROM feedback f                -- Alias 'f' pour la table de feedback (équivalent du 'NEW' actuel)
JOIN eleve_groupe eg           -- Alias 'eg' pour les groupes de l'utilisateur
  ON eg.num_etudiant = f.user_id    -- Jointure sur l'ID de l'utilisateur
ON CONFLICT DO NOTHING;


ALTER TABLE public.feedback DROP CONSTRAINT feedback_student_id_fkey;
ALTER TABLE public.feedback ADD CONSTRAINT feedback_student_id_fkey FOREIGN KEY (num_etudiant) REFERENCES public."user"(id) ON DELETE CASCADE;

ALTER TABLE public.user ADD COLUMN blame BOOLEAN DEFAULT FALSE;

ALTER TABLE public.student DROP CONSTRAINT elo_student_num_etudiant_fkey;
ALTER TABLE public.student ADD CONSTRAINT elo_student_num_etudiant_fkey FOREIGN KEY (num_etudiant) REFERENCES public."user"(id) ON DELETE CASCADE;
