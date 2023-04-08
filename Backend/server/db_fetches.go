package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func fetchRoomQuestions(db *sql.DB, accessCode string) error {
	var questions [10]int
	var code string
	log.Printf("Getting roomQuestions for room %s.\n", accessCode)
	query := "SELECT * FROM roomQuestions where accessCode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, accessCode)
	err = row.Scan(&code, &questions[0], &questions[1], &questions[2], &questions[3], &questions[4], &questions[5], &questions[6], &questions[7], &questions[8], &questions[9])
	if err != nil {
		log.Printf("There was an error while fetching Room Question IDs for Room %s, Error : %s\n", accessCode, err.Error())
		return err
	}
	for i := 0; i < 10; i++ {
		var question *Question
		question, err = fetchQuestion(db, questions[i])
		if err != nil {
			log.Printf("An error occurred while fetching question with id %d. Error : %s.", i, err.Error())
		} else {
			gameRooms[accessCode].questions[i] = &Question{question.ID, question.question, question.answer, question.option_1, question.option_2, question.option_3, question.option_4}
			log.Printf("Successfully assigned question with information to Game Room! : %d, %s, %s, %s, %s, %s, %s.\n", gameRooms[accessCode].questions[i].ID, gameRooms[accessCode].questions[i].question, gameRooms[accessCode].questions[i].answer, gameRooms[accessCode].questions[i].option_1, gameRooms[accessCode].questions[i].option_2, gameRooms[accessCode].questions[i].option_3, gameRooms[accessCode].questions[i].option_4)
		}
	}
	return nil
}
func fetchQuestion(db *sql.DB, ID int) (*Question, error) {
	log.Printf("Getting Question with ID : %d.\n", ID)
	query := "SELECT * FROM questions where question_id = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()
	var question Question
	row := stmt.QueryRowContext(ctx, ID)
	if err := row.Scan(&question.ID, &question.question, &question.answer, &question.option_1, &question.option_2, &question.option_3, &question.option_4); err != nil {
		log.Printf("There was an error while fetching Question with id %d, Error : %s\n", ID, err.Error())
		return nil, err
	} else {
		log.Printf("Successfully fetched question with information : %d, %s, %s, %s, %s, %s, %s.\n", question.ID, question.question, question.answer, question.option_1, question.option_2, question.option_3, question.option_4)
		return &question, nil
	}
	//var question_mem *Question
	//question_mem = new(Question)
	//(*question_mem).ID = question.ID
	//(*question_mem).question = question.question
	//(*question_mem).answer = question.answer
	//(*question_mem).option_1 = question.option_1
	//(*question_mem).option_2 = question.option_2
	//(*question_mem).option_3 = question.option_3
	//(*question_mem).option_4 = question.option_4

}

// Function used to fetch information about the game room : accessCode
// This function needs to be tested
func fetchRoom(db *sql.DB, accessCode string) error {
	log.Printf("Getting Game Room with access code : %s", accessCode)
	query := "select * from gameRoom WHERE accessCode = ?;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	var room gameRoom
	row := stmt.QueryRowContext(ctx, accessCode)
	err = row.Scan(&room.accessCode, &room.currentRound, &room.numOfPlayersAnswered, &room.numOfPlayersAnsweredCorrect, &room.numOfDisconnectedPlayers)
	if err != nil {
		log.Printf("There was an error while fetching game room with accessCode %s.", accessCode)
		return err
	}
	gameRooms[room.accessCode] = &gameRoom{accessCode: room.accessCode, currentRound: room.currentRound, numOfPlayersAnswered: room.numOfPlayersAnswered, numOfPlayersAnsweredCorrect: room.numOfPlayersAnsweredCorrect, numOfDisconnectedPlayers: room.numOfDisconnectedPlayers, questions: make(map[int]*Question), players: make(map[string]*roomUser)}
	log.Printf("Successfully loaded Game Room with information %s, %d, %d, %d, %d.\n", room.accessCode, gameRooms[room.accessCode].currentRound, gameRooms[room.accessCode].numOfPlayersAnswered, gameRooms[room.accessCode].numOfPlayersAnsweredCorrect, gameRooms[room.accessCode].numOfDisconnectedPlayers)
	return nil
}
func fetchRooms(db *sql.DB) error {
	log.Printf("Getting games")
	query := "select * from gameRoom;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var room gameRoom
		err := rows.Scan(&room.accessCode, &room.currentRound, &room.numOfPlayersAnswered, &room.numOfPlayersAnsweredCorrect, &room.numOfDisconnectedPlayers)
		if err != nil {
			return err
		}
		gameRooms[room.accessCode] = &gameRoom{accessCode: room.accessCode, currentRound: room.currentRound, numOfPlayersAnswered: room.numOfPlayersAnswered, numOfPlayersAnsweredCorrect: room.numOfPlayersAnsweredCorrect, numOfDisconnectedPlayers: room.numOfDisconnectedPlayers, questions: make(map[int]*Question), players: make(map[string]*roomUser)}
		log.Printf("Successfully loaded Game Room with information %s, %d, %d, %d, %d.\n", room.accessCode, gameRooms[room.accessCode].currentRound, gameRooms[room.accessCode].numOfPlayersAnswered, gameRooms[room.accessCode].numOfPlayersAnsweredCorrect, gameRooms[room.accessCode].numOfDisconnectedPlayers)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func fetchRoomUsers(db *sql.DB, accessCode string) error {
	log.Printf("Getting room Users for room %s.", accessCode)
	query := "select * FROM roomUser WHERE accessCode = ?;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx, accessCode)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rUser roomUser
		if err := rows.Scan(&rUser.username, &rUser.accessCode, &rUser.points, &rUser.ready, &rUser.offline, &rUser.roundAnswer, &rUser.correctAnswer); err != nil {
			return err
		}
		gameRooms[accessCode].players[rUser.username] = &roomUser{username: rUser.username, accessCode: rUser.accessCode, points: rUser.points, ready: rUser.ready, offline: rUser.offline, roundAnswer: rUser.roundAnswer, correctAnswer: rUser.correctAnswer}
		log.Printf("Successfully loaded user information :  %s, %s, %d, %d, %d, %d, %d.", gameRooms[accessCode].players[rUser.username].username, gameRooms[accessCode].players[rUser.username].accessCode, gameRooms[accessCode].players[rUser.username].points, gameRooms[accessCode].players[rUser.username].ready, gameRooms[accessCode].players[rUser.username].offline, gameRooms[accessCode].players[rUser.username].roundAnswer, gameRooms[accessCode].players[rUser.username].correctAnswer)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func fetchRoomUser(db *sql.DB, username string) (*roomUser, error) {
	log.Printf("Search for room user %s.\n", username)
	query := "SELECT * FROM roomUser where username = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()
	var player roomUser
	row := stmt.QueryRowContext(ctx, username)
	if err := row.Scan(&player.username, &player.accessCode, &player.points, &player.ready, &player.offline, &player.roundAnswer, &player.correctAnswer); err != nil {
		log.Printf("There was an error while fetching Room User %s, Error : %s\n", username, err.Error())
		return nil, err
	}
	// Need to test
	//var player_mem *roomUser
	//player_mem = new(roomUser)
	//(*player_mem).username = player.username
	//(*player_mem).accessCode = player.accessCode
	//(*player_mem).points = player.points
	//(*player_mem).ready = player.ready
	//(*player_mem).offline = player.offline
	//return player_mem, nil
	return &player, nil
}
