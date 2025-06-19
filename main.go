package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joshraphael/go-retroachievements"
	"github.com/joshraphael/go-retroachievements/models"
	"github.com/nao1215/markdown"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatalf("need at least one argument")
	}
	id := os.Args[1]
	gameID, err := strconv.Atoi(id)
	if err != nil {
		log.Fatalf("Argument must be an integer: %s", id)
	}
	secret := os.Getenv("RA_API_KEY")

	client := retroachievements.NewClient(secret)

	resp, err := client.GetGameExtended(models.GetGameExtentedParameters{
		GameID: gameID,
	})
	if err != nil {
		log.Fatalf("Error on RA call GetGameExtended with game id %d: %s", gameID, err.Error())
	}

	if resp == nil {
		log.Fatalf("No game found for id %d", gameID)
	}

	markdown.NewMarkdown(os.Stdout).
		H1(resp.Title).Build()
}
