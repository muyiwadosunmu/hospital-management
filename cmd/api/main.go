package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/muyiwadosunmu/hospital-management/internal/auth"
	store "github.com/muyiwadosunmu/hospital-management/internal/data"
	"github.com/muyiwadosunmu/hospital-management/internal/env"
	"github.com/muyiwadosunmu/hospital-management/internal/jsonlog"
	"github.com/muyiwadosunmu/hospital-management/internal/mailer"
)

const Version = "1.0.0"

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := config{
		port:   env.GetInt("PORT", 3000),
		addr:   env.GetString("ADDR", ":3000"),
		apiURL: env.GetString("SERVER_URL", "localhost:3000"),
		db: dbConfig{
			addr:         env.GetString("HOSPITAL_MGT_DSN", ""),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 25),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 25),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		env: env.GetString("ENV", "development"),
		mail: mailConfig{
			exp:      time.Hour * 24 * 3, //  3 days
			username: env.GetString("SMTP_USERNAME", "dosunmuoluwamuyiwa98@gmail.com"),
			password: env.GetString("SMTP_PASSWORD", "wchhdwrlijxlnilg"),
			host:     env.GetString("SMTP_HOST", "smtp.gmail.com"),
			port:     env.GetInt("SMTP_PORT", 465),
			sender:   env.GetString("SMTP_SENDER", "no-reply@struct.io"),
		},
		auth: authConfig{
			token: tokenConfig{
				secret: env.GetString("AUTH_TOKEN_SECRET", "qwertyuioplkjhg"),
				exp:    time.Hour * 24 * 3, // 3 days
				iss:    "gophersocial",
			},
		},
	}
	// Logger
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	go func() {

	}()
	// Database
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			logger.PrintError(err, nil)
		}
	}(db)

	logger.PrintInfo("Database Connection Pool Established", nil)

	app := &application{
		config:        cfg,
		models:        store.NewModels(db),
		logger:        logger,
		mailer:        mailer.New(cfg.mail.host, cfg.mail.port, cfg.mail.username, cfg.mail.password, cfg.mail.sender),
		authenticator: auth.NewJWTAuthenticator(cfg.auth.token.secret, cfg.auth.token.iss, cfg.auth.token.iss),

		// logger2: logger2,
	}
	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	// Use sql.Open() to create an empty connection pool, using the DSN from the config
	// struct.
	db, err := sql.Open("postgres", cfg.db.addr)
	if err != nil {

		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	// Use the time.ParseDuration() function to convert the idle timeout duration string
	// to a time.Duration type.
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	// Set the maximum idle timeout.
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Use PingContext() to establish a new connection to the database, passing in the
	// context we created above as a parameter. If the connection couldn't be
	// established successfully within the 5 second deadline, then this will return an
	// error.
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	// Return the sql.DB connection pool.
	return db, nil
}

// alias air  ='$(go env GOPATH)/bin/air'
// direnv allow .
// migrate create -seq -ext sql -dir ./cmd/migrate/migrations
// migrate -path ./cmd/migrate/migrations -database "postgres://admin:password@localhost:5432/gosocial?sslmode=disable" up
// migrate -path ./cmd/migrate/migrations -database "postgres://admin:password@localhost:5432/gosocial?sslmode=disable" down
// export PATH=$(go env GOPATH)/bin:$PATH
