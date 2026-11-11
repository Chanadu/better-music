package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/Chanadu/better-music/handlers"
)

func main() {
	handler := &handlers.Handler{}
	api := handler.RegisterAPI(http.NewServeMux())

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(api.OpenAPI()); err != nil {
		log.Fatal(err)
	}
}
