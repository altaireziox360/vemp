package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/lib/pq"
	"github.com/namikaze-dev/snippetbox/internal/models"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	snippets       models.SnippetModelInterface
	users          models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	// parse command-line flags
	addr := flag.String("addr", ":8000", "HTTP network address")
	dsn := flag.String("dsn", "postgres://postgres:yeager063x@localhost/snippetbox?sslmode=disable", "dsn for postgres db")
	flag.Parse()

	// create loggers
	infoLog := log.New(os.Stdout, "[INFO :]  ", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "[ERROR:]  ", log.Ldate|log.Ltime|log.Lshortfile)

	// connect db
	db, err := connectDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	infoLog.Println("connect db success")
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	// init session manager
	sessiongManager := scs.New()
	sessiongManager.Store = postgresstore.New(db)
	sessiongManager.Lifetime = 12 * time.Hour
	sessiongManager.Cookie.Secure = true

	// initialize app with dependencies
	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		snippets:       &models.SnippetModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    form.NewDecoder(),
		sessionManager: sessiongManager,
	}

	// serve app
	// tlsconfig := &tls.Config{
	// 	CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	// }

	srv := &http.Server{
		Addr:         *addr,
		Handler:      app.routes(),
		ErrorLog:     errorLog,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("listening on %s\n", *addr)
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}

func connectDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
