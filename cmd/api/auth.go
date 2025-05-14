package main

// import (
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"fmt"
// 	"net/http"
// 	"time"

// 	"github.com/golang-jwt/jwt/v5"
// 	"github.com/google/uuid"
// 	"github.com/muyiwadosunmu/hospital-management/internal/data"
// )

// type RegisterUserPayload struct {
// 	FirstName string `json:"firstName" validate:"required,max=100"`
// 	LastName  string `json:"lastName" validate:"required,max=100"`
// 	Email     string `json:"email" validate:"required,email,max=255"`
// 	Password  string `json:"password" validate:"required,min=3,max=72"`
// 	Role      string `json:"role" validate:"required,min=5, max=15"`
// }

// type CreateUserTokenPayload struct {
// 	Email    string `json:"email" validate:"required,email,max=255"`
// 	Password string `json:"password" validate:"required,min=3,max=72"`
// }

// type UserWithToken struct {
// 	*data.User
// 	Token string `json:"token"`
// }

// func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
// 	var payload RegisterUserPayload
// 	ctx := r.Context()

// 	if err := app.readJSON(w, r, &payload); err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}

// 	if err := Validate.Struct(payload); err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}

// 	user := &data.User{
// 		FirstName: payload.FirstName,
// 		LastName:  payload.LastName,
// 		Email:     payload.Email,
// 	}

// 	// hash the user password

// 	err := user.Password.Set(payload.Password)
// 	if err != nil {
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}

// 	// Store token in our database
// 	plainToken := uuid.New().String()
// 	hash := sha256.Sum256([]byte(plainToken))
// 	hashedToken := hex.EncodeToString(hash[:])

// 	// store the user
// 	err = app.models.Users.CreateAndInvite(ctx, user, hashedToken, app.config.mail.exp)
// 	if err != nil {
// 		switch err {
// 		case data.ErrDuplicateEmail:
// 			app.badRequestResponse(w, r, err)
// 		case data.ErrDuplicateUsername:
// 			app.badRequestResponse(w, r, err)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}

// 	// send email
// 	userWithToken := UserWithToken{
// 		User:  user,
// 		Token: plainToken,
// 	}

// 	app.background(func() {
// 		// As there are now multiple pieces of data that we want to pass to our email
// 		// templates, we create a map to act as a 'holding structure' for the data. This
// 		// contains the plaintext version of the activation token for the user, along
// 		// with their ID.
// 		data := map[string]interface{}{
// 			"activationToken": userWithToken.Token,
// 			"userID":          user.ID,
// 			"firstName":       userWithToken.FirstName,
// 			"lastName":        userWithToken.LastName,
// 			"activationURL":   fmt.Sprintf("%s/confirm/%s", app.config.frontendURL, plainToken),
// 		}

// 		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
// 		if err != nil {
// 			// Importantly, if there is an error sending the email then we use the
// 			// app.logger.PrintError() helper to manage it, instead of the
// 			// app.serverErrorResponse() helper like before.
// 			app.logger.PrintError(err, nil)
// 			if err := app.models.Users.Delete(ctx, user.ID); err != nil {
// 				app.logger.PrintError(err, map[string]string{"err": err.Error()})
// 			}
// 		}
// 	})

// 	if err := app.writeJSON(w, http.StatusCreated, envelope{"data": userWithToken}, nil); err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}

// }

// func (app *application) createTokenHandler(w http.ResponseWriter, r *http.Request) {
// 	var payload CreateUserTokenPayload
// 	if err := app.readJSON(w, r, &payload); err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}

// 	if err := Validate.Struct(payload); err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}

// 	user, err := app.models.Users.GetByEmail(r.Context(), payload.Email)
// 	app.logger.PrintInfo("user", map[string]string{"user": user.FirstName})
// 	if err != nil {
// 		switch err {
// 		case data.ErrRecordNotFound:
// 			app.unauthorizedErrorResponse(w, r, err)
// 		default:
// 			app.serverErrorResponse(w, r, err)
// 		}
// 		return
// 	}

// 	if err := user.Password.Compare(payload.Password); err != nil {
// 		app.unauthorizedErrorResponse(w, r, err)
// 		return
// 	}

// 	claims := jwt.MapClaims{
// 		"sub": user.ID,
// 		"exp": time.Now().Add(app.config.auth.token.exp).Unix(),
// 		"iat": time.Now().Unix(),
// 		"nbf": time.Now().Unix(),
// 		"iss": app.config.auth.token.iss,
// 		"aud": app.config.auth.token.iss,
// 	}

// 	token, err := app.authenticator.GenerateToken(claims)
// 	if err != nil {
// 		// app.logger.PrintInfo("err", map[string]string{"err": err.Error()})
// 		app.serverErrorResponse(w, r, err)
// 		return
// 	}

// 	if err := app.writeJSON(w, http.StatusCreated, envelope{
// 		"data": token,
// 	}, nil); err != nil {
// 		app.serverErrorResponse(w, r, err)
// 	}
// }
