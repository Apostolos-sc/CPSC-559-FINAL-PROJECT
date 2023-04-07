package server

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func updateRoom(db *sql.DB, room *gameRoom) error {
	query := "UPDATE gameRoom SET currentRound = ?, numOfPlayersAnswered=?, numOfPlayersAnsweredCorrect=?, numOfDisconnectedPlayers=? WHERE accesscode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, room.currentRound, room.numOfPlayersAnswered, room.numOfPlayersAnsweredCorrect, room.numOfDisconnectedPlayers, room.accessCode)
	if err != nil {
		log.Printf("Error %s when updating gameRoom table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d rows updated gameRoom now contains information %s, %d, %d, %d, %d.", rows, room.accessCode, room.currentRound, room.numOfPlayersAnswered, room.numOfPlayersAnsweredCorrect, room.numOfDisconnectedPlayers)
	return nil
}
func updateRoomUser(db *sql.DB, user *roomUser) error {
	query := "UPDATE roomUser SET points = ?, ready = ?, offline=?, roundAnswer=?, correctAnswer=? WHERE username = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, user.points, user.ready, user.offline, user.roundAnswer, user.correctAnswer, user.username)
	if err != nil {
		log.Printf("Error %s when updating roomUser table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d rows updated user now contains information %s, %s, %d, %d, %d, %d, %d.\n ", rows, user.username, user.accessCode, user.points, user.ready, user.offline, user.roundAnswer, user.correctAnswer)
	return nil
}
