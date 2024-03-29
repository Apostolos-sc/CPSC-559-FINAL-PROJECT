package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func fetchRoomQuestions(db1 *sql.DB, db2 *sql.DB, accessCode string) error {
    db := ping(db1, db2)
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
		question, err = fetchQuestion(db1, db2, questions[i])
		if err != nil {
			log.Printf("An error occurred while fetching question with id %d. Error : %s.", i, err.Error())
		} else {
			gameRooms[accessCode].questions[i+1] = &Question{ID: question.ID, question: question.question, answer: question.answer, option_1: question.option_1,option_2: question.option_2,option_3: question.option_3, option_4: question.option_4}
			log.Printf("Successfully assigned question with information to Game Room! : %d, %s, %s, %s, %s, %s, %s.\n", gameRooms[accessCode].questions[i+1].ID, gameRooms[accessCode].questions[i+1].question, gameRooms[accessCode].questions[i+1].answer, gameRooms[accessCode].questions[i+1].option_1, gameRooms[accessCode].questions[i+1].option_2, gameRooms[accessCode].questions[i+1].option_3, gameRooms[accessCode].questions[i+1].option_4)
		}
	}
	return nil
}
func fetchQuestion(db1 *sql.DB, db2 *sql.DB, ID int) (*Question, error) {
    db := ping(db1, db2)
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
func fetchRoom(db1 *sql.DB, db2 *sql.DB, accessCode string) error {
    db := ping(db1, db2)
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
	err = row.Scan(&room.accessCode, &room.currentRound, &room.numOfPlayersAnswered, &room.numOfPlayersAnsweredCorrect, &room.numOfDisconnectedPlayers, &room.accessCodeTimeStamp, &room.currentRoundTimeStamp, &room.numOfPlayersAnsweredTimeStamp, &room.numOfPlayersAnsweredCorrectTimeStamp, &room.numOfDisconnectedPlayersTimeStamp)
	if err != nil {
		log.Printf("There was an error while fetching game room with accessCode %s.", accessCode)
		return err
	}
	gameRooms[room.accessCode] = &gameRoom{accessCode: room.accessCode, currentRound: room.currentRound, numOfPlayersAnswered: room.numOfPlayersAnswered, numOfPlayersAnsweredCorrect: room.numOfPlayersAnsweredCorrect, numOfDisconnectedPlayers: room.numOfDisconnectedPlayers, accessCodeTimeStamp: room.accessCodeTimeStamp, currentRoundTimeStamp: room.currentRoundTimeStamp,numOfPlayersAnsweredTimeStamp: room.numOfPlayersAnsweredTimeStamp, numOfPlayersAnsweredCorrectTimeStamp: room.numOfPlayersAnsweredCorrectTimeStamp, numOfDisconnectedPlayersTimeStamp: room.numOfDisconnectedPlayersTimeStamp, questions: make(map[int]*Question), players: make(map[string]*roomUser)}
	log.Printf("Successfully loaded Game Room with information %s, %d, %d, %d, %d, %d, %d, %d, %d, %d.\n", room.accessCode, gameRooms[room.accessCode].currentRound, gameRooms[room.accessCode].numOfPlayersAnswered, gameRooms[room.accessCode].numOfPlayersAnsweredCorrect, gameRooms[room.accessCode].numOfDisconnectedPlayers, gameRooms[room.accessCode].accessCodeTimeStamp, gameRooms[room.accessCode].currentRoundTimeStamp, gameRooms[room.accessCode].numOfPlayersAnsweredTimeStamp, gameRooms[room.accessCode].numOfPlayersAnsweredCorrectTimeStamp, gameRooms[room.accessCode].numOfDisconnectedPlayersTimeStamp)
	return nil
}
func fetchRoomByTimeStamp(db1 *sql.DB, db2 *sql.DB, time_stamp int64) (*gameRoom, error) {
    db := ping(db1, db2)
	log.Printf("Getting Game Room with time stamp : %d", time_stamp)
	query := "select * from gameRoom WHERE accessCodeTimeStamp = ?;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()
	var room gameRoom
	row := stmt.QueryRowContext(ctx, time_stamp)
	err = row.Scan(&room.accessCode, &room.currentRound, &room.numOfPlayersAnswered, &room.numOfPlayersAnsweredCorrect, &room.numOfDisconnectedPlayers, &room.accessCodeTimeStamp, &room.currentRoundTimeStamp, &room.numOfPlayersAnsweredTimeStamp, &room.numOfPlayersAnsweredCorrectTimeStamp, &room.numOfDisconnectedPlayersTimeStamp)
	if err != nil {
		log.Printf("There was an error while fetching game room with time_stamp %d.", time_stamp)
		return nil, err
	}
	log.Printf("Successfully loaded Game Room with information %s, %d, %d, %d, %d, %d, %d, %d, %d, %d.\n", room.accessCode, room.currentRound, room.numOfPlayersAnswered, room.numOfPlayersAnsweredCorrect, room.numOfDisconnectedPlayers, room.accessCodeTimeStamp, room.currentRoundTimeStamp, room.numOfPlayersAnsweredTimeStamp, room.numOfPlayersAnsweredCorrectTimeStamp, room.numOfDisconnectedPlayersTimeStamp)
	return &gameRoom{accessCode: room.accessCode, currentRound: room.currentRound, numOfPlayersAnswered: room.numOfPlayersAnswered, numOfPlayersAnsweredCorrect: room.numOfPlayersAnsweredCorrect, numOfDisconnectedPlayers: room.numOfDisconnectedPlayers, accessCodeTimeStamp: room.accessCodeTimeStamp,currentRoundTimeStamp: room.currentRoundTimeStamp, numOfPlayersAnsweredTimeStamp: room.numOfPlayersAnsweredTimeStamp, numOfPlayersAnsweredCorrectTimeStamp: room.numOfPlayersAnsweredCorrectTimeStamp, numOfDisconnectedPlayersTimeStamp: room.numOfDisconnectedPlayersTimeStamp, questions: make(map[int]*Question), players: make(map[string]*roomUser)}, err
}
func fetchRooms(db1 *sql.DB, db2 *sql.DB,) error {
    db := ping(db1, db2)
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
		err := rows.Scan(&room.accessCode, &room.currentRound, &room.numOfPlayersAnswered, &room.numOfPlayersAnsweredCorrect, &room.numOfDisconnectedPlayers, &room.accessCodeTimeStamp, &room.currentRoundTimeStamp, &room.numOfPlayersAnsweredTimeStamp, &room.numOfPlayersAnsweredCorrectTimeStamp, &room.numOfDisconnectedPlayersTimeStamp)
		if err != nil {
			return err
		}
		gameRooms[room.accessCode] = &gameRoom{accessCode: room.accessCode, currentRound: room.currentRound, numOfPlayersAnswered: room.numOfPlayersAnswered, numOfPlayersAnsweredCorrect: room.numOfPlayersAnsweredCorrect, numOfDisconnectedPlayers: room.numOfDisconnectedPlayers, accessCodeTimeStamp: room.accessCodeTimeStamp,currentRoundTimeStamp: room.currentRoundTimeStamp, numOfPlayersAnsweredTimeStamp: room.numOfPlayersAnsweredTimeStamp, numOfPlayersAnsweredCorrectTimeStamp: room.numOfPlayersAnsweredCorrectTimeStamp, numOfDisconnectedPlayersTimeStamp: room.numOfDisconnectedPlayersTimeStamp, questions: make(map[int]*Question), players: make(map[string]*roomUser)}
		log.Printf("Successfully loaded Game Room with information %s, %d, %d, %d, %d, %d,%d, %d, %d, %d.\n", room.accessCode, gameRooms[room.accessCode].currentRound, gameRooms[room.accessCode].numOfPlayersAnswered, gameRooms[room.accessCode].numOfPlayersAnsweredCorrect, gameRooms[room.accessCode].numOfDisconnectedPlayers, gameRooms[room.accessCode].accessCodeTimeStamp, gameRooms[room.accessCode].currentRoundTimeStamp, gameRooms[room.accessCode].numOfPlayersAnsweredTimeStamp, gameRooms[room.accessCode].numOfPlayersAnsweredCorrectTimeStamp, gameRooms[room.accessCode].numOfDisconnectedPlayersTimeStamp)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func fetchRoomUsers(db1 *sql.DB, db2 *sql.DB, accessCode string) error {
    db := ping(db1, db2)
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
		if err := rows.Scan(&rUser.username, &rUser.accessCode, &rUser.points, &rUser.ready, &rUser.offline, &rUser.roundAnswer, &rUser.accessCodeTimeStamp,&rUser.correctAnswer, &rUser.pointsTimeStamp, &rUser.readyTimeStamp, &rUser.offlineTimeStamp, &rUser.roundAnswerTimeStamp, &rUser.correctAnswerTimeStamp); err != nil {
			return err
		}
		gameRooms[accessCode].players[rUser.username] = &roomUser{username: rUser.username, accessCode: rUser.accessCode, points: rUser.points, ready: rUser.ready, offline: rUser.offline, roundAnswer: rUser.roundAnswer, correctAnswer: rUser.correctAnswer, accessCodeTimeStamp: rUser.accessCodeTimeStamp ,pointsTimeStamp: rUser.pointsTimeStamp, readyTimeStamp: rUser.readyTimeStamp, offlineTimeStamp: rUser.offlineTimeStamp, roundAnswerTimeStamp: rUser.roundAnswerTimeStamp, correctAnswerTimeStamp: rUser.correctAnswerTimeStamp}
		log.Printf("Successfully loaded user information :  %s, %s, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d.", gameRooms[accessCode].players[rUser.username].username, gameRooms[accessCode].players[rUser.username].accessCode, gameRooms[accessCode].players[rUser.username].points, gameRooms[accessCode].players[rUser.username].ready, gameRooms[accessCode].players[rUser.username].offline, gameRooms[accessCode].players[rUser.username].roundAnswer, gameRooms[accessCode].players[rUser.username].correctAnswer, gameRooms[accessCode].players[rUser.username].accessCodeTimeStamp, gameRooms[accessCode].players[rUser.username].pointsTimeStamp, gameRooms[accessCode].players[rUser.username].readyTimeStamp, gameRooms[accessCode].players[rUser.username].offlineTimeStamp, gameRooms[accessCode].players[rUser.username].roundAnswerTimeStamp, gameRooms[accessCode].players[rUser.username].correctAnswerTimeStamp)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func fetchRoomUser(db1 *sql.DB, db2 *sql.DB, username string) (*roomUser, error) {
    db := ping(db1, db2)
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
	if err := row.Scan(&player.username, &player.accessCode, &player.points, &player.ready, &player.offline, &player.roundAnswer, &player.correctAnswer, &player.accessCodeTimeStamp ,&player.pointsTimeStamp, &player.readyTimeStamp, &player.offlineTimeStamp, &player.roundAnswerTimeStamp, &player.correctAnswerTimeStamp); err != nil {
		log.Printf("There was an error while fetching Room User %s, Error : %s\n", username, err.Error())
		return nil, err
	} else {
		log.Printf("Successfully fetched user information :  %s, %s, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d.", player.username, player.accessCode, player.points, player.ready, player.offline, player.roundAnswer, player.correctAnswer, player.accessCodeTimeStamp, player.pointsTimeStamp, player.readyTimeStamp, player.offlineTimeStamp, player.roundAnswerTimeStamp, player.correctAnswerTimeStamp)

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
