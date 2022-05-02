package main

import (
	"context"
	"github.com/jackc/pgx/v4"

	"encoding/json"

	"errors"

	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"io"

	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

var db *pgxpool.Pool

type data struct {
	Date     string  `json:"date"`
	Proteins float64 `json:"proteins"`
	Fats     float64 `json:"fats"`
	Carbs    float64 `json:"carbs"`
}

func main() {

	var err error
	const conString = "postgres://pfc:L0ktar0gar@127.0.0.1:5432/dpfc"
	db, err = pgxpool.Connect(context.Background(), conString)

	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	router := httprouter.New()

	router.POST("/plus", plusHandler)
	router.POST("/minus", minusHandler)
	router.GET("/get", getHandler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

func plusHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request, err := readRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO pfc (date, proteins, fats, carbs)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (date) DO UPDATE SET
		proteins = pfc.proteins + $2, fats = pfc.fats + $3, carbs = pfc.carbs + $4;`, request.Date, request.Proteins, request.Fats, request.Carbs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Возникла внутреняя ошибка сервера"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func minusHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request, err := readRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE pfc SET
		proteins = GREATEST(pfc.proteins - $2, 0), fats = GREATEST(pfc.fats - $3, 0), carbs = GREATEST(pfc.carbs - $4, 0)
		WHERE date = $1;`, request.Date, request.Proteins, request.Fats, request.Carbs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Возникла внутреняя ошибка сервера"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func getHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var response data

	response.Date = r.URL.Query().Get("date")
	if response.Date != "" {
		if err := validateDate(response.Date); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	} else {
		response.Date = currentDate()
	}

	err := db.QueryRow(context.Background(), "SELECT proteins, fats, carbs FROM pfc WHERE date=$1", response.Date).Scan(
		&response.Proteins,
		&response.Fats,
		&response.Carbs,
	)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
		return
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Возникла внутреняя ошибка сервера"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func readRequestBody(r *http.Request) (data, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return data{}, err
	}

	var requestBody data

	err = json.Unmarshal(body, &requestBody)
	if err != nil {
		return data{}, err
	}

	if requestBody.Date != "" {
		if err = validateDate(requestBody.Date); err != nil {
			return data{}, err
		}
	} else {
		requestBody.Date = currentDate()
	}

	return requestBody, nil
}

func validateDate(s string) error {
	_, err := time.Parse("2006-01-02", s)
	return err
}

func currentDate() string {
	year, month, day := time.Now().Date()

	return fmt.Sprintf("%d-%d-%d", year, int(month), day)
}
