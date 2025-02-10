package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	// "zombiezen.com/go/sqlite"

	"tfs/templates"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/d1"
	cfOption "github.com/cloudflare/cloudflare-go/v4/option"

	"github.com/joho/godotenv"

	"github.com/oklog/ulid/v2"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	// env setup
    path_dir := "./"
    if os.Getenv("MODE") == "PRODUCTION" {
        path_dir = "/app/"
    }
	err := godotenv.Load(filepath.Join(path_dir, ".env"))
	if err != nil {
        log.Print(err)
		log.Print("Error loading .env file")

        for true {

        }
	}

	// r2/d1 auth
	cfClient := cloudflare.NewClient(cfOption.WithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN")))

	// Check if can get d1
	db, err := cfClient.D1.Database.Get(
		context.TODO(),
		os.Getenv("D1_ID"),
		d1.DatabaseGetParams{
			AccountID: cloudflare.F(os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
		},
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("Loaded file db %+v\n", db.JSON.Name)

	// r2 using s3 api
	r2AkeyID := os.Getenv("R2_AKEYID")
	r2Secret := os.Getenv("R2_SECRET")
	r2Url := os.Getenv("R2_URL")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2AkeyID, r2Secret, "")),
		config.WithRegion("auto"),
	)
    if err != nil {
        log.Fatal(err)
    }

    r2Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(r2Url)
    })

    r2PresignClient := s3.NewPresignClient(r2Client)

	// actual REST routes
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	// panic/error recovery
	r.Use(middleware.Recoverer)

	// heartbeat endpoint for pinging
	r.Use(middleware.Heartbeat("/heartbeat"))

	// sentry.io
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(templates.IndexPage()).ServeHTTP(w, r)
	})

	// r2 session, id etc logging
	r.Get("/getUploadKey", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		// filesize := r.FormValue("filesize")
		// filename := r.FormValue("filename")

		tokenCookie, err := r.Cookie("token")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		token := tokenCookie.Value
		page, err := cfClient.D1.Database.Query(context.TODO(),
			os.Getenv("D1_ID"),
			d1.DatabaseQueryParams{
				AccountID: cloudflare.F(os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
				Sql:       cloudflare.F("SELECT * FROM tokens WHERE token = ?"),
				Params:    cloudflare.F([]string{token}),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Query Failed"))
			panic(err)
		}

		id := ulid.Make()
		unique := false
		for !unique {
			page, err = cfClient.D1.Database.Query(context.TODO(),
				os.Getenv("D1_ID"),
				d1.DatabaseQueryParams{
					AccountID: cloudflare.F(os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
					Sql:       cloudflare.F("SELECT * FROM files WHERE uuid = ?"),
					Params:    cloudflare.F([]string{id.String()}),
				},
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Query Failed"))
				panic(err)
			}

			if (*page)[0].Success && len((*page)[0].Results) == 0 {
				unique = true
			} else {
				id = ulid.Make()
			}
		}

        // presign the s3 url
        presignResult, err := r2PresignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
            Bucket: aws.String(os.Getenv("R2_BUCKETNAME")),
            Key: aws.String(id.String()),
        })

		page, err = cfClient.D1.Database.Query(context.TODO(),
			os.Getenv("D1_ID"),
			d1.DatabaseQueryParams{
				AccountID: cloudflare.F(os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
				Sql:       cloudflare.F("INSERT INTO files VALUES (?,?,?,?)"),
				Params:    cloudflare.F([]string{id.String(), r.FormValue("filename"), "r2", strconv.FormatInt(time.Now().Unix(), 10)}),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Query Failed"))
			panic(err)
		}

		if (*page)[0].Success {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(fmt.Sprintf(`{"method": "s3", "url": "%s", "id": "%s"}`, presignResult.URL, id.String())))
		} else {
            w.WriteHeader(http.StatusInternalServerError)
            w.Write([]byte("Something went wrong. Please try again"))
            panic((*page)[0].Results)
		}
	})

	auth_cache := map[string]struct{}{}
	last_cleared := time.Now()

	r.Post("/auth", func(w http.ResponseWriter, r *http.Request) {
		if time.Now().Sub(last_cleared) > 1e+9 {
			auth_cache = map[string]struct{}{}
			last_cleared = time.Now()
		} else if _, exists := auth_cache[r.RemoteAddr]; !exists {
			w.WriteHeader(http.StatusForbidden)
			return
		} else {
			auth_cache[r.RemoteAddr] = struct{}{}
		}

		r.ParseForm()
		token := r.FormValue("token")
		page, err := cfClient.D1.Database.Query(context.TODO(),
			os.Getenv("D1_ID"),
			d1.DatabaseQueryParams{
				AccountID: cloudflare.F(os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
				Sql:       cloudflare.F("SELECT * FROM tokens WHERE token = ?"),
				Params:    cloudflare.F([]string{token}),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Query Failed"))
			panic(err)
		}

		if (*page)[0].Success && len((*page)[0].Results) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid"))
		} else if (*page)[0].Success && len((*page)[0].Results) == 1 {
			tokenCookie := http.Cookie{
				Name:  "token",
				Value: token,
				Path:  "/",

				MaxAge:  2592000000,
				Expires: (time.Now()).AddDate(0, 1, 0),

				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			}

			http.SetCookie(w, &tokenCookie)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Authenticated!"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong."))
		}
	})

	fs := http.FileServer(http.Dir(filepath.Join(path_dir, "/static/")))
	r.Handle("/static/*", http.StripPrefix("/static", fs))

	http.ListenAndServe(":3000", r)
}
