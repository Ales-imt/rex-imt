package user

import (
	"back-rex-common/pkg/services"
	"context"
	"log"
	"net/http"
)

var UserContextKey = &services.ContextKey{Name: "LogEntry"}

func getUserFromCtx(r *http.Request) *User {
	user, ok := r.Context().Value(UserContextKey).(*User)
	if ok {
		return user
	}
	log.Fatal("utilisateur inconnu")
	return nil
}

func setUserFromCtx(r *http.Request, user *User) context.Context {
	return context.WithValue(r.Context(), UserContextKey, user)
}
