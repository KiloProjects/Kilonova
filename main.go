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

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/grader/judge"
	"github.com/KiloProjects/Kilonova/models"
	"github.com/KiloProjects/Kilonova/server"
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
		if err == nil {
			break
		}
	}
	fmt.Println("Connected to DB")

	db.AutoMigrate(&models.MOTD{})
	db.AutoMigrate(&models.EvalTest{})
	db.AutoMigrate(&models.Problem{})
	db.AutoMigrate(&models.Task{})
	db.AutoMigrate(&models.Test{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Limits{})

	manager = datamanager.NewManager("/data")

	// Initialize router
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

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	API := server.NewAPI(ctx, db, config, manager)
	grader := judge.NewGrader(ctx, db, manager)

	err = grader.NewManager(2)
	if err != nil {
		panic(err)
	}

	r.Mount("/api", API.GetRouter())
	grader.Start()

	// for graceful setup and shutdown
	server := &http.Server{Addr: "0.0.0.0:8080", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	cancel()
	fmt.Println("Shutting Down")
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
	db.Close()
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
