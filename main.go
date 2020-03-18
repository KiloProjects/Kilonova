package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/AlexVasiluta/kilonova/api"
	"github.com/AlexVasiluta/kilonova/eval"
	"github.com/AlexVasiluta/kilonova/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	db     *gorm.DB
	config *models.Config
)

func main() {
	var err error
	for {
		fmt.Println("Trying to connect to DB until it works")
		db, err = gorm.Open("mysql", "kilonova:kn_password@(db)/kilonova?charset=utf8&parseTime=True&loc=Local")
		if err == nil {
			break
		}
	}
	fmt.Println("Connected to DB")

	config, err = readConfig()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Read config")

	db.AutoMigrate(&models.MOTD{})
	db.AutoMigrate(&models.EvalTest{})
	db.AutoMigrate(&models.Problem{})
	db.AutoMigrate(&models.Task{})
	db.AutoMigrate(&models.Test{})
	db.AutoMigrate(&models.User{})

	os.MkdirAll("/app/knTests", 0777)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	frontend := api.NewAPI(ctx, db, config)

	go frontend.Run()
	go eval.StartEvalListener(ctx, db, config)

	// Setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	fmt.Println("Shutting Down")

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
