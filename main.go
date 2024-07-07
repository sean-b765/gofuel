package main

import (
	"fmt"
	"net/http"

	"example.com/fuel/routes"
	"github.com/gorilla/mux"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

var gorillaLambda *gorillamux.GorillaMuxAdapterV2

func Init() {
	r := mux.NewRouter()

	// 404 handler
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, fmt.Sprintf("Not found: [%s]\n", r.RequestURI), http.StatusNotFound)
	})

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK\n"))
	}).Methods("GET")

	r.HandleFunc("/nearest/{coordinates}", routes.GetNearest).Methods("GET")
	r.HandleFunc("/cheapest/{coordinates}", routes.GetCheapest).Methods("GET").Queries("radius", "{radius:[0-9]+}")

	// Error - missing radius query
	r.HandleFunc("/cheapest/{coordinates}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Missing ?radius={value} in URL.\n"))
	}).Methods("GET")

	// Create GorillaMuxAdapterV2 from router
	gorillaLambda = gorillamux.NewV2(r)
}

func main() {
	Init()
	lambda.Start(gorillaLambda.ProxyWithContext)
}
