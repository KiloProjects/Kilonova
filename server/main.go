package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/AlexVasiluta/kilonova/api"
	"github.com/AlexVasiluta/kilonova/datamanager"
	"github.com/AlexVasiluta/kilonova/eval"
	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	// _ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	db      *gorm.DB
	config  *models.Config
	manager *datamanager.Manager
)

func main() {

	config, err := readConfig()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Read config")

	fmt.Println("Trying to connect to DB until it works")
	for {
		db, err = gorm.Open("postgres", "sslmode=disable host=db user=kilonova password=kn_password dbname=kilonova")
		// db, err = gorm.Open("mysql", "kilonova:kn_password@(db)/kilonova?charset=utf8&parseTime=True&loc=Local")
		if err == nil {
			break
		}
	}
	fmt.Println("Connected to DB")

	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&models.MOTD{})
	db.AutoMigrate(&models.EvalTest{})
	db.AutoMigrate(&models.Problem{})
	db.AutoMigrate(&models.Task{})
	db.AutoMigrate(&models.Test{})
	db.AutoMigrate(&models.User{})

	manager = datamanager.NewManager("/data")

	r := chi.NewRouter()
	corsConfig := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	r.Use(corsConfig.Handler)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(20 * time.Second))
	r.Use(middleware.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	frontend := api.NewAPI(ctx, db, config, manager)

	r.Mount("/api", frontend.GetRouter())
	go eval.StartEvalListener(ctx, db, config, manager)

	// graceful setup and shutdown
	server := &http.Server{Addr: "0.0.0.0:8080", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	fmt.Println("Shutting Down")
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}

}

func readConfig() (*models.Config, error) {
	data, err := ioutil.ReadFile("/app/config.json")
	if err != nil {
		return nil, err
	}
	var config models.Config
	json.Unmarshal(data, &config)
	return &config, nil
}
