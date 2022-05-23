package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var db *pgxpool.Pool

type Data struct {
	Date     string  `json:"date"`
	Proteins float64 `json:"proteins"`
	Fats     float64 `json:"fats"`
	Carbs    float64 `json:"carbs"`
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	var err error
	const connString = "postgres://pfc:L0ktar0gar@127.0.0.1:5432/dpfc"

	db, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Print(err)
		return 1
	}
	defer db.Close()

	router := httprouter.New()
	router.POST("/pfc", handler(plusHandler))
	router.PATCH("/pfc", handler(minusHandler))
	router.GET("/pfc", handler(getHandler))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	if err = server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}

	return 0
}

type handlerFunc func(http.ResponseWriter, *http.Request, httprouter.Params) error

var ErrBadRequest = errors.New("bad request")
var ErrNotFound = errors.New("not found")

func handler(f handlerFunc) httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		err := f(writer, request, params)
		if err != nil {
			if errors.Is(err, ErrBadRequest) {
				writer.WriteHeader(http.StatusBadRequest)
			} else if errors.Is(err, pgx.ErrNoRows) {
				writer.WriteHeader(http.StatusNotFound)
			} else if errors.Is(err, ErrNotFound) {
				writer.WriteHeader(http.StatusNotFound)
			} else {
				writer.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	}
}

func plusHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	data, err := readBodyData(r)
	if err != nil {
		return err
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO pfc (date, proteins, fats, carbs)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (date) DO 
        UPDATE SET proteins = pfc.proteins + $2, fats = pfc.fats + $3, carbs = pfc.carbs + $4;`,
		data.Date, data.Proteins, data.Fats, data.Carbs)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusResetContent)

	return nil
}

func minusHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	data, err := readBodyData(r)
	if err != nil {
		return err
	}

	tag, err := db.Exec(context.Background(), `
		UPDATE pfc 
		SET proteins = GREATEST(pfc.proteins - $2, 0),
		fats = GREATEST(pfc.fats - $3, 0),
		carbs = GREATEST(pfc.carbs - $4, 0) 
		WHERE date = $1`,
		data.Date, data.Proteins, data.Fats, data.Carbs)
	if err != nil {
		return err
	}

	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}

func getHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	var response Data

	response.Date = r.URL.Query().Get("date")
	if response.Date != "" {
		if err := validateDate(response.Date); err != nil {
			return ErrBadRequest
		}
	} else {
		response.Date = currentDate()
	}

	err := db.QueryRow(context.Background(),
		"SELECT proteins, fats, carbs "+
			"FROM pfc "+
			"WHERE date=$1 "+
			"LIMIT 1", response.Date).Scan(
		&response.Proteins,
		&response.Fats,
		&response.Carbs,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else {
			return err
		}
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("Возникла внутреняя ошибка сервера")
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)

	return nil
}

func readBodyData(r *http.Request) (*Data, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var bodyData = &Data{}

	err = json.Unmarshal(body, &bodyData)
	if err != nil {
		return nil, err
	}

	if bodyData.Date != "" {
		if err = validateDate(bodyData.Date); err != nil {
			return nil, err
		}
	} else {
		bodyData.Date = currentDate()
	}

	return bodyData, nil
}

func validateDate(s string) error {
	_, err := time.Parse("2006-01-02", s)
	return err
}

func currentDate() string {
	year, month, day := time.Now().Date()
	return fmt.Sprintf("%d-%d-%d", year, int(month), day)
}
