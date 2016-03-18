package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/salsita/go-pivotaltracker.v1/v5/pivotal"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Errorf("Error loading .env file (the bot may not work).\n")
	}

	textToLookFor := os.Getenv("TEXT_TO_LOOK_FOR")
	textToReplaceWith := os.Getenv("TEXT_TO_REPLACE_WITH")
	name := os.Getenv("NAME")

	pivotalToken := os.Getenv("PIVOTAL_AUTH_TOKEN")
	pivotalClient := pivotal.NewClient(pivotalToken)
	pivotalProjectID, err := strconv.Atoi(os.Getenv("PIVOTAL_PROJECT_ID"))
	if err != nil {
		panic(fmt.Sprintf("Unable to parse pivotal project id.\nError: %v\n", pivotalProjectID))
	}

	for {
		// Get the stories updated over the last hour.
		stories, err := pivotalClient.Stories.List(pivotalProjectID, "updated:-1h")
		if err != nil {
			fmt.Errorf("Error retrieving pivotal stories.\nError: %v\n", err)
		}

		for _, story := range stories {
			fmt.Printf("Found story, id: %d, name: %s\n", story.Id, story.Name)

			comments, response, err := pivotalClient.Stories.ListComments(pivotalProjectID, story.Id)
			if err != nil {
				fmt.Errorf("Error retrieving pivotal comments for story %d.\nError: %v\nResponse: %v\n",
					story.Id, response, err)
			}

			var commentsContainTextToLookFor bool
			var commentsAlreadyHaveAutoGeneratedCorrection bool
			var commentToUpdate string
			for _, comment := range comments {
				fmt.Printf("Found comment, story id: %d, comment text: %s\n", story.Id, comment.Text)

				commentText := comment.Text

				// Check if the comment includes the words to be replaced.
				// Also check if the bot has already made the correction, if so
				// we proabably don't want to correct it again
				if strings.Contains(commentText, textToLookFor) {
					commentsContainTextToLookFor = true
					commentToUpdate = commentText
				} else if strings.Contains(commentText, "Auto Generated") &&
					strings.Contains(commentText, textToReplaceWith) {
					commentsAlreadyHaveAutoGeneratedCorrection = true
				}
			}

			if commentsContainTextToLookFor && !commentsAlreadyHaveAutoGeneratedCorrection {
				newComment := strings.Replace(commentToUpdate, textToLookFor, textToReplaceWith, 1)

				// Add new comment with the replace substring
				botComment := fmt.Sprintf("Auto Generated Comment:\n\nLooks like %s still doesn't know the correct link...\n\n%s", name, newComment)
				fmt.Printf("Posted comment %s for story id %d\n", botComment, story.Id)
				pivotalComment := pivotal.Comment{
					Text: botComment,
				}
				_, _, err := pivotalClient.Stories.AddComment(pivotalProjectID, story.Id, &pivotalComment)
				if err != nil {
					fmt.Errorf("Error adding new comment.\nError: %v\n", err)
				}
			}
		}

		time.Sleep(time.Minute * 30)
	}
}
