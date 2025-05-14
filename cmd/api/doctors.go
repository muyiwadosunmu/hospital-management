package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/muyiwadosunmu/hospital-management/internal/data"
)

// type userKey string

// const userCtx userKey = "user"

// type RegisterUserPayload struct {
// 	FirstName string `json:"firstName" validate:"required,max=100"`
// 	LastName  string `json:"lastName" validate:"required,max=100"`
// 	Email     string `json:"email" validate:"required,email,max=255"`
// 	Password  string `json:"password" validate:"required,min=3,max=72"`
// }

// type CreateUserTokenPayload struct {
// 	Email    string `json:"email" validate:"required,email,max=255"`
// 	Password string `json:"password" validate:"required,min=3,max=72"`
// }

func (app *application) registerDoctorHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	ctx := r.Context()

	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.Doctor{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
	}

	// hash the user password

	err := user.Password.Set(payload.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// store the user
	err = app.models.Doctors.Create(ctx, user)
	if err != nil {
		switch err {
		case data.ErrDuplicateEmail:
			app.badRequestResponse(w, r, err)
		case data.ErrDuplicateUsername:
			app.badRequestResponse(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// send welcome email
	app.background(func() {
		data := map[string]interface{}{
			"userID":    user.ID,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	if err := app.writeJSON(w, http.StatusCreated, envelope{"data": user}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) createDocTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.models.Doctors.GetDocByEmail(r.Context(), payload.Email)
	app.logger.PrintInfo("user", map[string]string{"user": user.FirstName})
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundRequestResponse(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := user.Password.Compare(payload.Password); err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(app.config.auth.token.exp).Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"iss": app.config.auth.token.iss,
		"aud": app.config.auth.token.iss,
	}

	token, err := app.authenticator.GenerateToken(claims)
	if err != nil {
		// app.logger.PrintInfo("err", map[string]string{"err": err.Error()})
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, envelope{
		"data": token,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getDocUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.getRecUser(r.Context(), userID)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundRequestResponse(w, r, err)
			return
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"data": user}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// func (app *application) userContextMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		userIdStr := chi.URLParam(r, "userId")
// 		userId, err := strconv.ParseInt(userIdStr, 10, 64)
// 		if err != nil {
// 			app.logger.PrintError(err, map[string]string{
// 				"error": "invalid user id format",
// 			})
// 			app.notFoundResponse(w, r)
// 			return
// 		}
// 		if userId < 1 {
// 			app.notFoundResponse(w, r)
// 			return
// 		}
// 		ctx := r.Context()

// 		user, err := app.models.Receptionists.GetById(ctx, userId)
// 		if err != nil {
// 			switch err {
// 			case data.ErrRecordNotFound:
// 				app.notFoundResponse(w, r)
// 				return
// 			default:
// 				app.serverErrorResponse(w, r, err)
// 			}
// 		}
// 		ctx = context.WithValue(ctx, userCtx, user)
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

func getDocUserFromContext(r *http.Request) *data.Doctor {
	user, _ := r.Context().Value(userCtx).(*data.Doctor)
	return user
}
