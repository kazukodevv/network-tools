package main

import (
	"time"

	"httpclient"
)

func main() {
	httpclient.New(httpclient.Config{
		BaseURL: "https://jsonplaceholder.typicode.com",
		Timeout: 10 * time.Second,
		Headers: map[string]string{
			"User-Agent": "MyApp/1.0",
		},
	})
}
