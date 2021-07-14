package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var randomNameBytes int = 8
var passPhrase string
var storagePath string
var publicRoot string
var maxUploadSize int64 = 100 << 20 // 200 mb
var htmlBox packr.Box = packr.NewBox("./html")

func main() {
	passPhrase = os.Getenv("UPLOAD_KEY")
	if passPhrase == "" || passPhrase == "DEFAULT_KEY" {
		log.Printf("UPLOAD_KEY environment variable can't be 'DEFAULT' or empty, exiting...")
		os.Exit(1)
	}

	storagePath = os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		log.Printf("STORAGE_PATH environment variable wasn't set, exiting...")
		os.Exit(2)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		log.Printf("Couldn't get working directory, exiting...")
		os.Exit(3)
	}
	storagePath = path.Join(workingDir, storagePath)
	writeTestFilename := "/write-test.txt"
	dst, err := os.Create(storagePath + writeTestFilename)
	if err != nil {
		log.Printf("STORAGE_PATH = %s is not writeable, exiting...", storagePath)
		os.Exit(3)
	}
	dst.Close()
	os.Remove(storagePath + writeTestFilename)

	publicRoot = os.Getenv("PUBLIC_ROOT")
	if publicRoot == "" {
		log.Printf("PUBLIC_ROOT environment variable wasn't set, exiting...")
		os.Exit(4)
	}

	randomNameBytesUInt64, err := strconv.ParseUint(os.Getenv("FILENAME_LENGTH"), 10, 32)
	if err != nil {
		log.Printf("FILENAME_LENGTH environment variable wasn't set, exiting...")
		os.Exit(5)
	}
	randomNameBytes = int(randomNameBytesUInt64)

	maxUploadSizeMB, err := strconv.ParseInt(os.Getenv("MAX_UPLOAD_SIZE_IN_MB"), 10, 64)
	if err != nil {
		log.Printf("MAX_UPLOAD_SIZE_IN_MB environment variable wasn't set, exiting...")
		os.Exit(6)
	}
	maxUploadSize = maxUploadSizeMB << 20

	listenAddr := os.Getenv("LISTEN_ADDRESS")
	if listenAddr == "" {
		log.Printf("LISTEN_ADDRESS environment variable wasn't set, exiting...")
		os.Exit(7)
	}

	enableWebform := os.Getenv("ENABLE_WEBFORM") == "true"

	router := mux.NewRouter()

	if enableWebform {
		router.HandleFunc("/", renderStatic("index.html", "text/html")).
			Methods("GET")

		router.HandleFunc("/style.css", renderStatic("style.css", "text/css")).
			Methods("GET")

		router.HandleFunc("/scripts.js", renderStatic("scripts.js", "text/javascript")).
			Methods("GET")
	}

	router.HandleFunc("/", uploadHandler()).
		Methods("POST", "PUT")

	router.PathPrefix("/").
		Methods("GET").
		Handler(http.StripPrefix("/", http.FileServer(http.Dir(storagePath))))

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	srv.Handler = handlers.LoggingHandler(os.Stdout, srv.Handler)

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", srv.ReadTimeout, "Grace period for existing connections to finish")
	flag.Parse()

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // SIGINT / SIGTERM

	log.Printf("Listening at %s", srv.Addr)
	log.Printf("Web interface is %s", map[bool]string{true: "enabled", false: "disabled"}[enableWebform])
	log.Printf("Public root set to %s", publicRoot)
	log.Printf("Password is %s", passPhrase)
	log.Printf("Storage is at %s", storagePath)
	log.Printf("Max file size is %v MB", maxUploadSize>>20)

	<-c // Block until SIGINT / SIGTERM

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	srv.Shutdown(ctx)

	log.Println("Shutting down...")

	// <-ctx.Done()

	os.Exit(0)
}

func renderStatic(staticFileName string, mimeType string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		staticDataBytes, err := htmlBox.Find(staticFileName)

		if err != nil {
			renderError(w, "Static file not found", http.StatusNotFound)
		} else {
			w.Header().Set("content-type", mimeType)
			w.Write(staticDataBytes)
		}
	})
}

func uploadHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const maxBufferSize = 1 << 20 // 1MB

		keyAccepted := false
		uploadOk := false
		redirect := true

		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for {
			part, err := reader.NextPart()

			if err == io.EOF {
				break
			}

			if part.FormName() == "key" {
				formKeyValueBuffer := new(bytes.Buffer)
				formKeyValueBuffer.ReadFrom(part)

				if formKeyValueBuffer.String() != passPhrase {
					http.Error(w, "Bad key", http.StatusUnauthorized)
					return
				} else {
					keyAccepted = true
					continue
				}
			}

			if part.FormName() == "noredirect" {
				redirect = false
			}

			if part.FileName() == "" { // if part.FileName() is empty, skip this iteration.
				continue
			}

			if !keyAccepted {
				http.Error(w, "No key provided as first field", http.StatusUnauthorized)
				return // abort if no key has been processed yet
			}

			newFileName := makeRandomFileName(randomNameBytes, filepath.Ext(part.FileName()))
			newFilePath := filepath.Join(storagePath, newFileName)

			dst, err := os.Create(newFilePath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			transferBuffer := make([]byte, maxBufferSize)
			writtenBytes, err := io.CopyBuffer(dst, part, transferBuffer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("Uploaded %s, %v KB", newFileName, writtenBytes>>10)

			if redirect {
				http.Redirect(w, r, publicRoot+newFileName, http.StatusFound)
			} else {
				render200(w, publicRoot+newFileName, http.StatusOK)
			}

			uploadOk = true
		}

		if !uploadOk {
			http.Error(w, "No file uploaded", http.StatusBadRequest)
		}
	})
}

func makeRandomFileName(randomBytes int, suffix string) string {
	b := make([]byte, randomBytes)
	rand.Read(b)
	return fmt.Sprintf("%x", b) + suffix
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

func render200(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message))
}
