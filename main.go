package main

import (
	"log"

	"dashyreborn/internal/app"
)

var appVersion = "dev"

func main() {
	if err := app.Run(appVersion); err != nil {
		log.Fatal(err)
	}
}
