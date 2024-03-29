package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func insertRoomUser(db1 *sql.DB, db2 *sql.DB, player *roomUser) error {
	db := ping(db1, db2)
	query := "INSERT INTO roomUser(username, accessCode, points, ready, offline, roundAnswer, correctAnswer, accessCodeTimeStamp, pointsTimeStamp, readyTimeStamp, offlineTimeStamp, roundAnswerTimeStamp, correctAnswerTimeStamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, player.username, player.accessCode, player.points, player.ready, player.offline, player.roundAnswer, player.correctAnswer, player.accessCodeTimeStamp, player.pointsTimeStamp, player.readyTimeStamp, player.offlineTimeStamp, player.roundAnswerTimeStamp, player.correctAnswerTimeStamp)
	if err != nil {
		log.Printf("Error %s when inserting row into roomUser table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d roomUser created with information %s, %s, %d, %d,  %d, %d, %d, %d, %d, %d,  %d, %d, %d. ", rows, player.username, player.accessCode, player.points, player.ready, player.offline, player.roundAnswer, player.correctAnswer, player.accessCodeTimeStamp, player.pointsTimeStamp, player.readyTimeStamp, player.offlineTimeStamp, player.roundAnswerTimeStamp, player.correctAnswerTimeStamp)
	return nil
}

func insertGameRoom(db1 *sql.DB, db2 *sql.DB, room *gameRoom) error {
	db := ping(db1, db2)
	query := "INSERT INTO gameRoom(accessCode, currentRound, numOfPlayersAnswered, numOfPlayersAnsweredCorrect, numOfDisconnectedPlayers, accessCodeTimeStamp, currentRoundTimeStamp, numOfPlayersAnsweredTimeStamp, numOfPlayersAnsweredCorrectTimeStamp, numOfDisconnectedPlayersTimeStamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, room.accessCode, room.currentRound, room.numOfPlayersAnswered, room.numOfPlayersAnsweredCorrect, room.numOfDisconnectedPlayers, room.accessCodeTimeStamp, room.currentRoundTimeStamp, room.numOfPlayersAnsweredTimeStamp, room.numOfPlayersAnsweredCorrectTimeStamp, room.numOfDisconnectedPlayersTimeStamp)
	if err != nil {
		log.Printf("Error %s when inserting row into gameRoom table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d gameRoom created with information %s, %d, %d, %d, %d, %d, %d, %d, %d, %d.\n", rows, room.accessCode, room.currentRound, room.numOfPlayersAnswered, room.numOfPlayersAnsweredCorrect, room.numOfDisconnectedPlayers, room.accessCodeTimeStamp, room.currentRoundTimeStamp, room.numOfPlayersAnsweredTimeStamp, room.numOfPlayersAnsweredCorrectTimeStamp, room.numOfDisconnectedPlayersTimeStamp)
	return nil
}

func insertRoomQuestions(db1 *sql.DB, db2 *sql.DB, accessCode string, q1 int, q2 int, q3 int, q4 int, q5 int, q6 int, q7 int, q8 int, q9 int, q10 int) error {
	db := ping(db1, db2)
	query := "INSERT INTO roomQuestions(accessCode, question_1_id,question_2_id,question_3_id,question_4_id,question_5_id,question_6_id,question_7_id,question_8_id,question_9_id,question_10_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10)
	if err != nil {
		log.Printf("Error %s when inserting row into roomQuestions table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d roomQuestions created with information %s, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d.", rows, accessCode, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10)
	return nil
}
