-- name: CreateStudent :exec
INSERT INTO student (user_id)
	VALUES (@user_id);

-- name: IsStudentExist :one
SELECT EXISTS (SELECT 1 FROM student WHERE user_id = @user_id);

-- name: CreateUser :one
INSERT INTO public.user (email, name, surname, roles, auth_source)
	VALUES ( @email, @name, @surname,@roles, @auth_source) RETURNING id;