package server

import (
	"database/sql"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

func generateAccessCode() string {
	//write code that generates access code, must be unique
	for {
		response, err := http.Get("https://www.random.org/integers?num=1&min=0&max=10000000&col=2&base=10&format=plain&rnd=new")

		if err != nil {
			log.Printf("There was an error while connecting to generate accessCode. Error: %s.\n", err.Error())
			os.Exit(1)
		}

		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Access Code was uniquely generated : %s", string(responseData))
		_, ok := gameRooms[strings.TrimSpace(string(responseData))]
		if ok {
			log.Printf("Room with access Code : %s exists. Will generate another one!.\n", strings.TrimSpace(string(responseData)))
		} else {
			log.Printf("Room with access Code : %s was generated!.\n", strings.TrimSpace(string(responseData)))
			return strings.TrimSpace(string(responseData))
		}

	}
}

func generateQuestions(db *sql.DB, accessCode string) bool {
	rand.Seed(time.Now().UnixNano())

	// Intn generates a random integer between 0 and 100
	// (not including 100)
	//read from DB the largest ID. and use that.
	var randomInts [10]int
	var maxQuestionsID int = 1428
	var questionsList [10]*Question
	for i := 0; i < 10; i++ {
		randomInts[i] = rand.Intn(maxQuestionsID)
		log.Printf("Random Questions ID %d for Room :%s generated.", randomInts[i], accessCode)
	}
	err := insertRoomQuestions(db, accessCode, randomInts[0], randomInts[1], randomInts[2], randomInts[3], randomInts[4], randomInts[5], randomInts[6], randomInts[7], randomInts[8], randomInts[9])
	if err != nil {
		//log the error return false
		log.Printf("There was an error when inserting the random generated Questions for room with accessCode : %s. Error : %s.\n", accessCode, err.Error())
		return false
	} else {
		for i := 0; i < 10; i++ {
			questionsList[i], err = fetchQuestion(db, randomInts[i])
			if err != nil {
				log.Printf("There was an issue when reading Question %d for game Room : %s. Error : %s.\n", i, accessCode, err.Error())
			} else {
				//assign from 1 to 10, 1 question for each round!
				gameRooms[accessCode].questions[i+1] = questionsList[i]
			}
		}
		//return true if we successfully added them to the database
		return true
	}
}
