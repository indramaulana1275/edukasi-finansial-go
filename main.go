package main

import (
	"edukasi-finansial-goedukasi-finansial-go/bootstrap"
)

func main() {
	app := bootstrap.Boot()

	app.Start()
}
