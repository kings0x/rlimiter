package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kings0x/rlimiter/engine"
	"github.com/kings0x/rlimiter/middleware"
	"github.com/kings0x/rlimiter/requestlimiter"
	"github.com/redis/go-redis/v9"
)

func ExampleClient() *redis.Client {
	url := "your redis url"
	opts, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	return redis.NewClient(opts)
}

func main() {

	rdb := ExampleClient()

	reqLimiter := requestlimiter.New(requestlimiter.Options{
		Rate:     1,
		Capacity: 5,
		Name:     "tokenbucket-req",
		Store:    requestlimiter.NewRedisStore(rdb),
	})

	e := engine.New(reqLimiter)

	mw := middleware.New(e, nil)

	mux := http.NewServeMux()

	mux.Handle("/", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK %s\n", time.Now().Format(time.RFC3339))
	})))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}

//http://localhost:8080
