-- name: CreateStudent :exec
INSERT INTO student (user_id, elo, last_elo_gain, promotion)
	VALUES (@user_id,  0, 0, @promotion);

-- name: IsStudentExist :one
SELECT EXISTS (SELECT 1 FROM student WHERE user_id = @user_id);

-- name: CreateUser :one
INSERT INTO public.user (email, name, surname, roles)
	VALUES ( @email, @name, @surname,@roles) RETURNING id;