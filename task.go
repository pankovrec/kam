package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
)

func getTask(TaskID int) (Task, error) {
	//Retrieve
	res := Task{
		ID:                TaskID,
		User:              "",
		Userid:            0,
		TaskDescription:   "",
		PublicationDate:   time.Time{},
		FinishDate:        time.Time{},
		Duration:          0,
		Places:            "",
		Placeid:           0,
		Comments:          "",
		Lenta:             false,
		CompletedByPetrov: false,
		CompletedByPankov: false,
	}

	var id int
	var user string
	var user_id int
	var description string
	var places string
	var place_id int
	var publicationDate pq.NullTime
	var finishDate pq.NullTime
	var duration pq.NullTime
	var comments string
	var lenta bool
	var completedByPetrov bool
	var completedByPankov bool

	err := db.QueryRow(`SELECT tasks.id, concat(users.surname, ' ', users.name) as user, user_id, description, places.name as place, place_id, publication_date, finish_date, duration, comments, lenta, completed_by_petrov, completed_by_pankov FROM tasks join users on tasks.user_id=users.id join places on tasks.place_id=places.id where tasks.id = $1`, TaskID).Scan(&id, &user, &user_id, &description, &places, &place_id, &publicationDate, &finishDate, &duration, &comments, &lenta, &completedByPetrov, &completedByPankov)
	if err == nil {
		res = Task{ID: id, User: user, Userid: user_id, TaskDescription: description, Places: places, Placeid: place_id, PublicationDate: publicationDate.Time, FinishDate: finishDate.Time, Comments: comments, Lenta: lenta, CompletedByPetrov: completedByPetrov, CompletedByPankov: completedByPankov}
		res.Duration = finishDate.Time.Sub(res.PublicationDate)
	}
	return res, err
}

func allTasks(startDate, endDate time.Time) ([]Task, error) {
	Tasks := []Task{}

	rows, err := db.Query(`
			SELECT tasks.id, concat(users.surname, ' ', users.name) as name, 
				   description, places.name as pages, 
				   publication_date, finish_date, duration, comments, lenta, completed_by_petrov, completed_by_pankov 
			FROM tasks 
			JOIN users ON tasks.user_id=users.id 
			JOIN places ON tasks.place_id=places.id 
			WHERE publication_date >= $1 AND publication_date <= $2
			ORDER BY tasks.id DESC`,
		startDate, endDate)

	for rows.Next() {
		var id int
		var user string
		var description string
		var places string
		var publicationDate pq.NullTime
		var finishDate pq.NullTime
		var duration pq.NullTime
		var comments string
		var lenta bool
		var completedByPetrov bool
		var completedByPankov bool

		err = rows.Scan(&id, &user, &description, &places, &publicationDate, &finishDate, &duration, &comments, &lenta, &completedByPetrov, &completedByPankov)
		if err != nil {
			return Tasks, err
		}

		currentTask := Task{ID: id, User: user, TaskDescription: description, Places: places, Comments: comments, Lenta: lenta, CompletedByPetrov: completedByPetrov, CompletedByPankov: completedByPankov}
		if publicationDate.Valid {
			currentTask.PublicationDate = publicationDate.Time
		}
		if finishDate.Valid {
			currentTask.FinishDate = finishDate.Time
		}
		currentTask.Duration = finishDate.Time.Sub(currentTask.PublicationDate)

		Tasks = append(Tasks, currentTask)
	}

	return Tasks, err
}

func getAllUsers() ([]UserList, error) {
	//Retrieve
	Users := []UserList{}

	rows, err := db.Query(`SELECT id, surname, name from users where invisible=false`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var surname string

		err = rows.Scan(&id, &name, &surname)
		if err != nil {
			return Users, err
		}

		listUsers := UserList{ID: id, Name: name, Surname: surname}

		Users = append(Users, listUsers)
	}

	return Users, err
}

func getAllPlaces() ([]PlacesList, error) {
	//Retrieve
	Places := []PlacesList{}

	rows, err := db.Query(`SELECT id, name from places`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string

		err = rows.Scan(&id, &name)
		if err != nil {
			return Places, err
		}

		listPlaces := PlacesList{ID: id, Name: name}

		Places = append(Places, listPlaces)
	}
	return Places, err

}

func getUserId(task_id int) (int, error) {
	var user_id int

	err := db.QueryRow(`select user_id from tasks where id = $1`, task_id).Scan(&user_id)
	if err != nil {
		return 0, err
	}
	return user_id, nil
}

func getUserIdFromName(user string) (int, error) {

	split := strings.Split(user, " ")
	var user_id int

	err := db.QueryRow(`select id from users where name = $1 and surname = $2`, split[1], split[0]).Scan(&user_id)
	if err != nil {
		return 0, err
	}
	return user_id, nil
}

func getPlaceIdFromName(name string) (int, error) {
	var place_id int

	err := db.QueryRow(`select id from places where name = $1`, name).Scan(&place_id)
	if err != nil {
		return 0, err
	}
	return place_id, nil
}

func getPlaceId(task_id int) (int, error) {
	var place_id int

	err := db.QueryRow(`select place_id from tasks where id = $1`, task_id).Scan(&place_id)
	if err != nil {
		return 0, err
	}
	return place_id, nil
}

func insertTask(user int, description string, place_id int,
	publicationDate, finishDate time.Time,
	comments string, lenta, completedByPetrov, completedByPankov bool) (int, error) {
	var TaskID int
	dur := finishDate.Sub(publicationDate)
	duration := time.Duration.Seconds(dur)

	// Добавляем контекст для лучшего контроля
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.QueryRowContext(ctx, `
INSERT INTO tasks(user_id, description, place_id, 
			 publication_date, finish_date, 
			 duration, comments, lenta, completed_by_petrov, completed_by_pankov) 
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
RETURNING id`,
		user, description, place_id,
		publicationDate, finishDate,
		duration, comments, lenta, completedByPetrov, completedByPankov).Scan(&TaskID)

	if err != nil {
		return 0, fmt.Errorf("ошибка создания задачи: %v", err)
	}

	log.Printf("Создана новая задача с ID: %d", TaskID)
	return TaskID, nil
}

func updateTask(id int, user int, description string, place_id int, publicationDate time.Time, finishDate time.Time, comments string, lenta bool, completedByPetrov bool, completedByPankov bool) (int, error) {
	//Create
	dur := finishDate.Sub(publicationDate)
	duration := time.Duration.Seconds(dur)
	res, err := db.Exec(`UPDATE tasks set user_id=$1, description=$2, place_id=$3, publication_date=$4, finish_date=$5, duration=$6, comments=$7, lenta=$8, completed_by_petrov=$9, completed_by_pankov=$10 where id=$11 RETURNING id`, user, description, place_id, publicationDate, finishDate, duration, comments, lenta, completedByPetrov, completedByPankov, id)
	if err != nil {
		return 0, err
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsUpdated), err
}

func removeTask(TaskID int) (int, error) {
	//Delete
	res, err := db.Exec(`delete from tasks where id = $1`, TaskID)
	if err != nil {
		return 0, err
	}

	rowsDeleted, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsDeleted), nil
}
