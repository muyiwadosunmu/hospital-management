package main

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	// httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	// r.Use(middleware.Compress(5, "application/json"))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Route("/api/v1", func(r chi.Router) {
		// r.With(app.BasicAuthMiddleware()).Get("/health", app.healthCheckHandler)
		docsURL := fmt.Sprintf("%s/swagger/doc.json", app.config.addr)
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL(docsURL), //The url pointing to API definition
		))
		r.Route("/doctors", func(r chi.Router) {
			r.Use(app.AuthDocTokenMiddleware)
			r.Get("/patients", app.getPatientsHandler)
			r.Route("/patients/{patientId}", func(r chi.Router) {
				r.Use(app.patientContextMiddleware)
				r.Get("/", app.getPatientDocHandler)
				r.Patch("/", app.updatePatientDocHandler)
			})
		})
		r.Route("/receptionists", func(r chi.Router) {
			r.Use(app.AuthRecTokenMiddleware)
			r.Get("/patients", app.getPatientsHandler)
			r.Post("/patients", app.registerPatientHandler)
			r.Route("/patients/{patientId}", func(r chi.Router) {
				r.Use(app.patientContextMiddleware)
				r.Get("/", app.getPatientHandler)
				r.Patch("/", app.updatePatientHandler)
				r.Delete("/", app.deletePatientHandler)
			})
			r.Group(func(r chi.Router) {
				r.Use(app.AuthRecTokenMiddleware)

			})

		})
		r.Route("/auth", func(r chi.Router) {
			r.Post("/receptionists", app.registerReceptionistHandler)
			r.Post("/receptionists/token", app.createRecTokenHandler)
			r.Post("/doctors", app.registerDoctorHandler)
			r.Post("/doctors/token", app.createDocTokenHandler)
		})

	})
	return r

}
