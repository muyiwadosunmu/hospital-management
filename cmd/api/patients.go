package main

import (
	"errors"
	"net/http"

	"github.com/muyiwadosunmu/hospital-management/internal/data"
)

type patientKey string

const patientCtx patientKey = "patient"

func (app *application) registerPatientHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	ctx := r.Context()
	receptionist := getRecUserFromContext(r)
	// fmt.Println(receptionist)

	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.Patient{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
		AddedBy:   receptionist,
	}

	// hash the user password

	err := user.Password.Set(payload.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// store the user
	err = app.models.Patients.CreatePatient(ctx, user)
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

func (app *application) deletePatientHandler(w http.ResponseWriter, r *http.Request) {
	post := getPatientFromCtx(r)
	ctx := r.Context()

	err := app.models.Patients.Delete(ctx, post.ID)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"data": "Patient deleted successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getPatientHandler(w http.ResponseWriter, r *http.Request) {

	patient := getPatientFromCtx(r)

	if err := app.writeJSON(w, http.StatusOK, envelope{"data": patient}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getPatientDocHandler(w http.ResponseWriter, r *http.Request) {

	patient := getPatientFromCtx(r)
	patient, err := app.models.Patients.GetDocPatientById(r.Context(), patient.ID)

	if err = app.writeJSON(w, http.StatusOK, envelope{"data": patient}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updatePatientHandler(w http.ResponseWriter, r *http.Request) {
	patient := getPatientFromCtx(r)
	ctx := r.Context()

	var payload struct {
		FirstName *string `json:"firstName" validate:"required,min=2,max=100"`
		LastName  *string `json:"lastName" validate:"required,min=2,max=1000"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if payload.FirstName != nil {
		patient.FirstName = *payload.FirstName
	}
	// We also do the same for the other fields in the input struct.
	if payload.LastName != nil {
		patient.LastName = *payload.LastName
	}

	err = app.models.Patients.UpdatePatient(ctx, patient)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"data": patient}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) updatePatientDocHandler(w http.ResponseWriter, r *http.Request) {
	patient := getPatientFromCtx(r)
	ctx := r.Context()

	var payload struct {
		FirstName *string     `json:"firstName" validate:"required,min=2,max=100"`
		LastName  *string     `json:"lastName" validate:"required,min=2,max=1000"`
		Data      interface{} `json:"data"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if payload.FirstName != nil {
		patient.FirstName = *payload.FirstName
	}
	// We also do the same for the other fields in the input struct.
	if payload.LastName != nil {
		patient.LastName = *payload.LastName
	}

	if payload.Data != nil {
		patient.Data = payload.Data
	}

	err = app.models.Patients.UpdatePatientByDoc(ctx, patient)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"data": patient}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func getPatientFromCtx(r *http.Request) *data.Patient {
	patient, _ := r.Context().Value(patientCtx).(*data.Patient)
	return patient
}
