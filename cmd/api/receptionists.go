package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/muyiwadosunmu/hospital-management/internal/data"
	"github.com/muyiwadosunmu/hospital-management/internal/validator"
)

type userKey string

const userCtx userKey = "user"

type RegisterUserPayload struct {
	FirstName string `json:"firstName" validate:"required,max=100"`
	LastName  string `json:"lastName" validate:"required,max=100"`
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required,min=3,max=72"`
}

type CreateUserTokenPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
}

func (app *application) getPatientsHandler(w http.ResponseWriter, r *http.Request) {
	var queryDto struct {
		FirstName string
		LastName  string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()
	ctx := r.Context()

	queryDto.FirstName = app.readString(qs, "firstName", "")
	queryDto.LastName = app.readString(qs, "lastName", "")

	queryDto.Page = app.readInt(qs, "page", 1, v)
	queryDto.PageSize = app.readInt(qs, "page_size", 10, v)

	queryDto.Sort = app.readString(qs, "sort", "id")

	// Add the supported sort values for this endpoint to the sort safelist.
	queryDto.SortSafelist = []string{"id", "firstName", "lastName", "-id", "-firstName", "-lastName"}

	if data.ValidateFilters(v, queryDto.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	posts, metadata, err := app.models.Patients.Get(ctx, queryDto.FirstName,
		queryDto.LastName, queryDto.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"data": posts, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) registerReceptionistHandler(w http.ResponseWriter, r *http.Request) {
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

	user := &data.Receptionist{
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
	err = app.models.Receptionists.Create(ctx, user)
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

func (app *application) createRecTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.models.Receptionists.GetByEmail(r.Context(), payload.Email)
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

func (app *application) getRecUserHandler(w http.ResponseWriter, r *http.Request) {
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

func getRecUserFromContext(r *http.Request) *data.Receptionist {
	user, _ := r.Context().Value(userCtx).(*data.Receptionist)
	return user
}
