package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var db *sql.DB

func main() {
	var (
		dsn = os.Getenv("dsn")
	)
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalln(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}

	router := gin.New()

	router.GET("/plus", plusHandler)
	router.GET("/minus", minusHandler)
	router.GET("/get", getHandler)

	if err := router.Run(":8080"); err != nil {
		log.Fatalln(err)
	}
}
func plusHandler(c *gin.Context) {
	date, proteins, fats, carbs, err := parseURL(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	_, err = db.Exec(`
    	INSERT INTO pfc (data, proteins, fats, carbs)
    
    	VALUES ($1, $2, $3, $4)
    
		ON CONFLICT (data) DO UPDATE SET
    
    	proteins = pfc.proteins + $2, fats = pfc.fats + $3, carbs = pfc.carbs + $4;`, date, proteins, fats, carbs)
	if err != nil {
		c.String(http.StatusInternalServerError, "Возникла внутреняя ошибка сервера")
		return
	}

	c.String(http.StatusOK, "ok")
}

func minusHandler(c *gin.Context) {
	date, proteins, fats, carbs, err := parseURL(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	_, err = db.Exec(`
    	UPDATE pfc SET
    
    	proteins = GREATEST(pfc.proteins - $2, 0), fats = GREATEST(pfc.fats - $3, 0), carbs = GREATEST(pfc.carbs - $4, 0)
    
    	WHERE data = $1;`, date, proteins, fats, carbs)
	if err != nil {
		c.String(http.StatusInternalServerError, "Возникла внутреняя ошибка сервера")
		return
	}

	c.String(http.StatusOK, "ok")
}

type response struct {
	Date     string  `json:"date"`
	Proteins float64 `json:"proteins"`
	Fats     float64 `json:"fats"`
	Carbs    float64 `json:"carbs"`
}

func getHandler(c *gin.Context) {
	var (
		resp response
		err  error
	)

	resp.Date, err = parseDate(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	err = db.QueryRow("SELECT proteins, fats, carbs FROM pfc WHERE data=$1", resp.Date).Scan(
		&resp.Proteins,
		&resp.Fats,
		&resp.Carbs,
	)
	if err != nil {
		log.Println(err)
		c.String(http.StatusInternalServerError, "Возникла внутреняя ошибка сервера")
		return
	}

	c.JSON(http.StatusOK, &resp)
}

var (
	ErrInvalidDate     = errors.New("Неправильная дата")
	ErrInvalidProteins = errors.New("неправильное значение белков")
	ErrInvalidFats     = errors.New("Неправильное значение жиров")
	ErrInvalidCarbs    = errors.New("Неправильное значение углеводов")
)

func parseDate(c *gin.Context) (date string, err error) {
	var (
		dateFromURL = c.Query("date")
		dateTime    time.Time
	)

	if dateFromURL != "" {
		dateTime, err = time.Parse("2006-01-02", dateFromURL)
		if err != nil {
			return date, ErrInvalidDate
		}
	} else {
		dateTime = time.Now()
	}
	year, month, day := dateTime.Date()

	date = fmt.Sprintf("%v-%v-%v", year, int(month), day)
	return date, nil
}

func parseURL(c *gin.Context) (date string, proteins, fats, carbs float64, err error) {
	date, err = parseDate(c)
	var (
		proteinsFromURL string = c.Query("proteins")
		fatsFromURL     string = c.Query("fats")
		carbsFromURL    string = c.Query("carbs")
	)

	if proteinsFromURL != "" {
		proteins, err = strconv.ParseFloat(proteinsFromURL, 64)
		if err != nil {
			return date, proteins, fats, carbs, ErrInvalidProteins
		}
	}

	if fatsFromURL != "" {
		fats, err = strconv.ParseFloat(fatsFromURL, 64)
		if err != nil {
			return date, proteins, fats, carbs, ErrInvalidFats
		}
	}

	if carbsFromURL != "" {
		carbs, err = strconv.ParseFloat(carbsFromURL, 64)
		if err != nil {
			return date, proteins, fats, carbs, ErrInvalidCarbs
		}
	}

	return date, proteins, fats, carbs, nil
}
