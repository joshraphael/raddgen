package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
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

	codeNotes, err := client.GetCodeNotes(models.GetCodeNotesParameters{
		GameID: gameID,
	})
	if err != nil {
		log.Fatalf("Error on RA call GetCodeNotes with game id %d: %s", gameID, err.Error())
	}

	if codeNotes == nil || !codeNotes.Success {
		log.Fatalf("No code notes found for id %d", gameID)
	}

	achievementTableOfContents := []string{}
	sortedAchievements := []models.GetGameExtentedAchievement{}

	for i := range resp.Achievements {
		achievement := resp.Achievements[i]
		sortedAchievements = append(sortedAchievements, achievement)
	}

	sort.Slice(sortedAchievements, func(i, j int) bool {
		return sortedAchievements[i].DisplayOrder < sortedAchievements[j].DisplayOrder
	})

	for i := range sortedAchievements {
		achievement := sortedAchievements[i]
		achievementTableOfContents = append(achievementTableOfContents, markdown.Link(fmt.Sprintf("%s (Achievement %d)", achievement.Title, achievement.ID), fmt.Sprintf("#achievement-%d", achievement.ID)))
		downloadImage(achievement.BadgeName, "out/badges")
	}

	codeNotesTableOfContents := []string{}
	for i := range codeNotes.CodeNotes {
		codeNote := codeNotes.CodeNotes[i]
		codeNotesTableOfContents = append(codeNotesTableOfContents, markdown.Link(fmt.Sprintf("Code Note %s", codeNote.Address), fmt.Sprintf("#code-note-%s", codeNote.Address)))
	}

	doc := markdown.NewMarkdown(os.Stdout)
	doc.H1f("Design Doc for %s", resp.Title)
	doc.H2("Table of Contents")
	doc.OrderedList([]string{
		markdown.Link("About", "#about"),
		markdown.Link("Learnings", "#learnings"),
		markdown.Link("Code Notes", "#code-notes"),
		markdown.Link("Achievements", "#achievements"),
		markdown.Link("Rich Presence", "#rich-presence"),
		markdown.Link("Leaderboards", "#leaderboards"),
	}...)
	doc.H2("About")
	doc.PlainTextf("<sub>%s</sub>", markdown.Link("Back to Table of Contents", "#table-of-contents"))
	doc.H2("Code Notes")
	doc.PlainTextf("<sub>%s</sub>", markdown.Link("Back to Table of Contents", "#table-of-contents"))
	doc.H3("Code Notes Navigation")
	doc.OrderedList(codeNotesTableOfContents...)
	for i := range codeNotes.CodeNotes {
		codeNote := codeNotes.CodeNotes[i]
		doc.H3f("Code Note %s", codeNote.Address)
		doc.PlainTextf("<sub>%s</sub><br>", markdown.Link("Back to navigation", "#code-notes-navigation"))
		doc.PlainTextf("<br>Author: %s<br>", markdown.Link(codeNote.User, fmt.Sprintf("https://retroachievements.org/user/%s", codeNote.User)))
		doc.PlainTextf("```txt\n%s\n```", codeNote.Note)
	}
	doc.H2("Achievements")
	doc.PlainTextf("<sub>%s</sub>", markdown.Link("Back to Table of Contents", "#table-of-contents"))
	doc.H3("Achievements Navigation")
	doc.OrderedList(achievementTableOfContents...)
	for i := range sortedAchievements {
		achievement := sortedAchievements[i]
		addAchievement(doc, achievement)
	}
	// Do stuff here
	doc.Build()
}

func addAchievement(doc *markdown.Markdown, achievement models.GetGameExtentedAchievement) {
	doc.H3f(markdown.Link(fmt.Sprintf("Achievement %d", achievement.ID), fmt.Sprintf("https://retroachievements.org/achievement/%d", achievement.ID)))
	doc.PlainTextf("<sub>%s</sub><br>", markdown.Link("Back to navigation", "#achievements-navigation"))
	doc.PlainTextf("<br>Title: %s", markdown.Bold(achievement.Title))
	doc.PlainTextf("<br>Author: %s", markdown.Link(achievement.Author, fmt.Sprintf("https://retroachievements.org/user/%s", achievement.Author)))
	if achievement.Type != "" {
		doc.PlainTextf("<br>Type: %s", markdown.BoldItalic(achievement.Type))
	}
	doc.PlainText(fmt.Sprintf("<br>Points: %s", markdown.Bold(fmt.Sprintf("%d", achievement.Points))))
	doc.PlainText(fmt.Sprintf("<br>%s<br>", markdown.Image(achievement.Title, fmt.Sprintf("badges/%s.png", achievement.BadgeName))))
	doc.PlainText(achievement.Description)
}

func downloadImage(imageID, downloadPath string) (err error) {
	file_name := fmt.Sprintf("%s/%s.png", downloadPath, imageID)
	if _, err := os.Stat(file_name); os.IsNotExist(err) {
		url := fmt.Sprintf("https://media.retroachievements.org/Badge/%s.png", imageID)
		head, e := http.Head(url)
		if e != nil {
			return err
		}
		defer head.Body.Close()
		if head.StatusCode == http.StatusForbidden {
			return nil
		}
		if head.StatusCode == http.StatusOK {
			response, e := http.Get(url)
			if e != nil {
				return err
			}
			defer response.Body.Close()
			if response.StatusCode == http.StatusOK {
				err := os.MkdirAll(downloadPath, 0777)
				if err != nil {
					return err
				}
				//open a file for writing
				file, err := os.Create(file_name)
				if err != nil {
					return err
				}
				defer file.Close()

				// Use io.Copy to just dump the response body to the file. This supports huge files
				_, err = io.Copy(file, response.Body)
				if err != nil {
					return err
				}
				return nil
			}
			return fmt.Errorf("unknown response from GET %s, got: %d", url, response.StatusCode)
		}
		return fmt.Errorf("unknown response from HEAD %s, got: %d", url, head.StatusCode)
	}
	return err
}
