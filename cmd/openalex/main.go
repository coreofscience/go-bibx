package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/coreofscience/go-bibx/clients"
)

func main() {
	openalexClient := clients.NewOpenAlexClient(&clients.NewOpenAlexClientParams{})
	works, err := openalexClient.ListRecentArticles(
		context.Background(),
		&clients.ListRecentArticlesParams{
			Query: "bit patterned media",
			Limit: nil,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	worksJSON, err := json.MarshalIndent(works, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(worksJSON))
}
