package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/muyiwadosunmu/hospital-management/internal/data"
	store "github.com/muyiwadosunmu/hospital-management/internal/data"
)

func (app *application) AuthRecTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is missing"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is malformed"))
			return
		}

		token := parts[1]
		jwtToken, err := app.authenticator.ValidateToken(token)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		claims, _ := jwtToken.Claims.(jwt.MapClaims)

		userID, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["sub"]), 10, 64)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}
		fmt.Println(userID, "Auth Token Middleware")

		ctx := r.Context()

		user, err := app.getRecUser(ctx, userID)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx = context.WithValue(ctx, userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) AuthDocTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is missing"))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is malformed"))
			return
		}

		token := parts[1]
		jwtToken, err := app.authenticator.ValidateToken(token)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		claims, _ := jwtToken.Claims.(jwt.MapClaims)

		userID, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["sub"]), 10, 64)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}
		fmt.Println(userID, "Auth Token Middleware")

		ctx := r.Context()

		user, err := app.getDocUser(ctx, userID)
		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx = context.WithValue(ctx, userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// func (app *application) postsContextMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		idParam := chi.URLParam(r, "postId")
// 		id, err := strconv.ParseInt(idParam, 10, 64)
// 		if err != nil {
// 			app.notFoundResponse(w, r)
// 			return
// 		}
// 		if id < 1 {
// 			app.notFoundResponse(w, r)
// 			return
// 		}

// 		ctx := r.Context()

// 		post, err := app.models.Posts.GetById(ctx, id)
// 		if err != nil {
// 			switch {
// 			case errors.Is(err, data.ErrRecordNotFound):
// 				app.notFoundRequestResponse(w, r, err)
// 			default:
// 				app.serverErrorResponse(w, r, err)
// 			}
// 			return
// 		}

// 		ctx = context.WithValue(ctx, postCtx, post)
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

func (app *application) patientContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIdStr := chi.URLParam(r, "patientId")
		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			app.logger.PrintError(err, map[string]string{
				"error": "invalid user id format",
			})
			app.notFoundResponse(w, r)
			return
		}
		if userId < 1 {
			app.notFoundResponse(w, r)
			return
		}
		ctx := r.Context()

		// Check if the request is coming from a doctor's route
		isDoctorRoute := strings.Contains(r.URL.Path, "/doctors/")

		var user *data.Patient
		var err2 error

		if isDoctorRoute {
			// Use GetDocPatientById for doctors to get the data field
			user, err2 = app.models.Patients.GetDocPatientById(ctx, userId)
		} else {
			// Use GetPatientById for receptionists to get basic info
			user, err2 = app.models.Patients.GetPatientById(ctx, userId)
		}

		if err2 != nil {
			switch err2 {
			case data.ErrRecordNotFound:
				app.notFoundResponse(w, r)
				return
			default:
				app.serverErrorResponse(w, r, err2)
			}
		}

		ctx = context.WithValue(ctx, patientCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) getRecUser(ctx context.Context, userID int64) (*store.Receptionist, error) {

	user, err := app.models.Receptionists.GetById(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (app *application) getDocUser(ctx context.Context, userID int64) (*store.Doctor, error) {

	user, err := app.models.Doctors.GetById(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (app *application) getPatient(ctx context.Context, userID int64) (*store.Patient, error) {
	user, err := app.models.Patients.GetPatientById(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// func (app *application) checkPostOwnership(requiredRole string, next http.HandlerFunc) http.HandlerFunc {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		user := getUserFromContext(r)
// 		post := getPostFromCtx(r)

// 		if post.UserID == user.ID {
// 			next.ServeHTTP(w, r)
// 			return
// 		}

// 		allowed, err := app.checkRolePrecedence(r.Context(), user, requiredRole)
// 		if err != nil {
// 			app.serverErrorResponse(w, r, err)
// 			return
// 		}

// 		if !allowed {
// 			app.forbiddenResponse(w, r, err)
// 			return
// 		}

// 		next.ServeHTTP(w, r)
// 	})
// }

// func (app *application) checkRolePrecedence(ctx context.Context, user *store.Receptionist, roleName string) (bool, error) {
// 	role, err := app.models.Roles.GetByName(ctx, roleName)
// 	if err != nil {
// 		return false, err
// 	}

// 	return user.Role.Level >= role.Level, nil
// }
