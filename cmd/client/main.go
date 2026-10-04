package main

import (
	"log"
	"net/http"
	"net/http/cookiejar"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.New()
	w := a.NewWindow("Task Manager")
	w.Resize(fyne.NewSize(800, 600))

	jar, jarErr := cookiejar.New(nil)

	if jarErr != nil {
		log.Fatal(jarErr)
	}

	client := http.Client{
		Jar: jar,
	}

	api := Api{
		client: &client,
	}

	w.ShowAndRun()
}
