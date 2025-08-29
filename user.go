package main

import (
	"fmt"
	"time"

	"github.com/lib/pq"
)

func getUser(UserID int) (User, error) {
	//Retrieve
	res := User{
		ID:         UserID,
		Name:       "",
		Surname:    "",
		Patronymic: "",
		Birthday:   time.Time{},
		Jobtitle:   "",
	}

	var id int
	var name string
	var surname string
	var patronymic string
	var birthday time.Time
	var jobtitle string

	err := db.QueryRow(`SELECT id, name, surname, patronymic, birthday, jobtitle FROM users where id = $1 and invisible=false`, UserID).Scan(&id, &name, &surname, &patronymic, &birthday, &jobtitle)
	if err == nil {
		res = User{ID: id, Name: name, Surname: surname, Patronymic: patronymic, Birthday: birthday, Jobtitle: jobtitle}
	}

	return res, err
}

func allUsers() ([]User, error) {
	//Retrieve
	Users := []User{}

	rows, err := db.Query(`SELECT id, name, surname, patronymic, birthday, jobtitle FROM users where invisible=false order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var surname string
		var patronymic string
		var birthday pq.NullTime
		var jobtitle string

		err = rows.Scan(&id, &name, &surname, &patronymic, &birthday, &jobtitle)
		if err != nil {
			return Users, err
		}

		currentUser := User{ID: id, Name: name, Surname: surname, Patronymic: patronymic, Jobtitle: jobtitle}
		currentUser.Birthday = birthday.Time

		Users = append(Users, currentUser)
	}

	return Users, err
}

func insertUser(name, surname string, patronymic string, birthday time.Time, jobtitle string) (int, error) {
	//Create
	var UserID int

	err := db.QueryRow(`INSERT INTO users(name, surname, patronymic, birthday, jobtitle) VALUES($1, $2, $3, $4, $5) RETURNING id`, name, surname, patronymic, birthday, jobtitle).Scan(&UserID)

	if err != nil {
		return 0, err
	}

	fmt.Printf("Last inserted User ID: %v\n", UserID)
	return UserID, err
}

func updateUser(id int, name, surname string, patronymic string, birthday time.Time, jobtitle string) (int, error) {

	res, err := db.Exec(`UPDATE users set name=$1, surname=$2, patronymic=$3, birthday=$4, jobtitle=$5 where id=$6 RETURNING id`, name, surname, patronymic, birthday, jobtitle, id)
	if err != nil {
		return 0, err
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsUpdated), err
}

func removeUser(UserID int) (int, error) {
	//Delete
	res, err := db.Exec(`delete from users where id = $1`, UserID)
	if err != nil {
		return 0, err
	}

	rowsDeleted, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsDeleted), nil
}
