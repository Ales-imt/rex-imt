-- name: CreateStudent :exec
INSERT INTO student (user_id, promotion)
	VALUES (@user_id, @promotion);

-- name: IsStudentExist :one
SELECT EXISTS (SELECT 1 FROM student WHERE user_id = @user_id);

-- name: CreateUser :one
INSERT INTO public.user (email, name, surname, roles)
	VALUES ( @email, @name, @surname,@roles) RETURNING id;