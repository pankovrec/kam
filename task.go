package main

import (
	"context"
	"database/sql"
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

func getKanbanTasks() ([]KanbanTask, error) {
	tasks := []KanbanTask{}

	rows, err := db.Query(`
        SELECT 
            t.id,
            t.description as title,
            CONCAT(u.surname, ' ', u.name) as user,
            t.user_id,
            t.description,
            t.publication_date as start_date,
            t.finish_date as deadline,
            COALESCE(t.status, 'queue') as status,
            t.lenta as urgent,
            COALESCE(t.progress, 0) as progress,
            COALESCE(t.paused, false) as paused,
            t.pause_until,
            t.pause_reason,
            t.comments
        FROM tasks t
        JOIN users u ON t.user_id = u.id
        WHERE t.is_kanban = true
        ORDER BY t.id DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var task KanbanTask
		var pauseUntil sql.NullTime
		var pauseReason sql.NullString
		var status string // ← Временная переменная для статуса

		err := rows.Scan(
			&task.ID, &task.Title, &task.User, &task.UserID,
			&task.Description, &task.StartDate, &task.Deadline,
			&status, // ← Сканируем во временную переменную
			&task.Urgent, &task.Progress, &task.Paused,
			&pauseUntil, &pauseReason, &task.Comments,
		)
		if err != nil {
			return nil, err
		}
		// Преобразуем статус
		if status == "progress" {
			task.Status = "in-progress"
		} else {
			task.Status = status
		}
		if pauseUntil.Valid {
			task.PauseUntil = pauseUntil.Time
		}
		if pauseReason.Valid {
			task.PauseReason = pauseReason.String
		}

		// Загружаем материалы для задачи
		materials, err := getTaskMaterials(task.ID)
		if err == nil {
			task.Materials = materials
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func getTaskMaterials(taskID int) ([]MaterialUsage, error) {
	materials := []MaterialUsage{}

	rows, err := db.Query(`
        SELECT 
            c.name as category,
            w.name as model,
            wh.qty as quantity
        FROM warehouse_history wh
        JOIN warehouse w ON wh.item_id = w.id
        JOIN categories c ON w.categorie_id = c.id
        WHERE wh.task_id = $1
    `, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var material MaterialUsage
		err := rows.Scan(&material.Category, &material.Model, &material.Quantity)
		if err != nil {
			return nil, err
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func insertKanbanTask(task KanbanTaskRequest) (int, error) {
	log.Printf("Создание задачи: %+v", task)

	deadline, err := time.Parse("2006-01-02T15:04", task.Deadline)
	if err != nil {
		log.Printf("Ошибка парсинга времени: %v", err)
		return 0, fmt.Errorf("неверный формат времени дедлайна: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return 0, err
	}
	defer tx.Rollback()

	var taskID int
	duration := deadline.Sub(time.Now())

	log.Printf("Вставка задачи в БД")
	err = tx.QueryRow(`
	    INSERT INTO tasks (
	        user_id, place_id, description,
	        publication_date, finish_date, duration,
	        comments, lenta, is_kanban, status
	    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9)
	    RETURNING id
	`, task.UserID, task.PlaceID, task.Description,
		time.Now(), deadline, duration.Seconds(),
		task.Comments, task.Urgent, task.Status).Scan(&taskID)

	if task.Materials != nil && len(task.Materials) > 0 {
		log.Printf("Обработка %d материалов", len(task.Materials))

		for i, material := range task.Materials {
			// Пропускаем пустые материалы
			if material.Category == "" || material.Model == "" || material.Quantity <= 0 {
				log.Printf("Пропускаем материал %d: пустые поля", i)
				continue
			}

			log.Printf("Обработка материала %d: %s - %s (%d шт.)", i, material.Category, material.Model, material.Quantity)

			// Находим ID материала
			var itemID int
			err := tx.QueryRow(`
                SELECT w.id 
                FROM warehouse w 
                JOIN categories c ON w.categorie_id = c.id 
                WHERE w.name = $1 AND c.name = $2
            `, material.Model, material.Category).Scan(&itemID)

			if err != nil {
				log.Printf("Материал не найден: %s - %s, ошибка: %v", material.Category, material.Model, err)
				return 0, fmt.Errorf("материал не найден: %s - %s", material.Category, material.Model)
			}

			log.Printf("Найден ID материала: %d", itemID)

			// Проверяем доступное количество
			var availableQty int
			err = tx.QueryRow(`
                SELECT qty FROM warehouse WHERE id = $1
            `, itemID).Scan(&availableQty)

			if err != nil {
				log.Printf("Ошибка проверки количества: %v", err)
				return 0, fmt.Errorf("ошибка проверки доступности материала: %v", err)
			}

			log.Printf("Доступное количество: %d, требуется: %d", availableQty, material.Quantity)

			if availableQty < material.Quantity {
				log.Printf("Недостаточно материала: доступно %d, требуется %d", availableQty, material.Quantity)
				return 0, fmt.Errorf("недостаточно материала: %s (доступно: %d)", material.Model, availableQty)
			}

			// Уменьшаем количество на складе
			_, err = tx.Exec(`
                UPDATE warehouse SET qty = qty - $1 WHERE id = $2
            `, material.Quantity, itemID)
			if err != nil {
				log.Printf("Ошибка обновления склада: %v", err)
				return 0, fmt.Errorf("ошибка списания материала: %v", err)
			}

			// Записываем в историю
			_, err = tx.Exec(`
                INSERT INTO warehouse_history (item_id, task_id, qty, operation_date)
                VALUES ($1, $2, $3, NOW())
            `, itemID, taskID, material.Quantity)
			if err != nil {
				log.Printf("Ошибка записи в историю: %v", err)
				return 0, fmt.Errorf("ошибка записи истории материала: %v", err)
			}

			log.Printf("Материал успешно обработан")
		}
	} else {
		log.Printf("Материалы не указаны, пропускаем обработку")
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Ошибка коммита транзакции: %v", err)
		return 0, fmt.Errorf("ошибка сохранения задачи: %v", err)
	}

	log.Printf("Задача успешно создана и сохранена")
	return taskID, nil
}

func getMaterialInfo(categoryName, modelName string) (int, int, error) {
	var itemID, availableQty int
	err := db.QueryRow(`
        SELECT w.id, w.qty
        FROM warehouse w
        JOIN categories c ON w.categorie_id = c.id
        WHERE c.name = $1 AND w.name = $2
    `, categoryName, modelName).Scan(&itemID, &availableQty)

	return itemID, availableQty, err
}
func updateKanbanTaskStatus(taskID int, status string) error {
	_, err := db.Exec(`
        UPDATE tasks 
        SET status = $1, 
            paused = false,
            pause_until = NULL,
            pause_reason = NULL
        WHERE id = $2 AND is_kanban = true
    `, status, taskID)
	return err
}

func pauseKanbanTask(taskID int, pauseUntil time.Time, pauseReason string) error {

	log.Printf("Приостановка до (местное): %v", pauseUntil)

	_, err := db.Exec(`
		  UPDATE tasks 
		  SET status = 'waiting',
			  paused = true,
			  pause_until = $1,
			  pause_reason = $2
		  WHERE id = $3 AND is_kanban = true
	  `, pauseUntil, pauseReason, taskID)
	return err

}

func getKanbanUsers() ([]UserResponse, error) {
	users := []UserResponse{}

	rows, err := db.Query(`
        SELECT id, name, surname 
        FROM users 
        WHERE invisible = false 
        ORDER BY surname, name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user UserResponse
		err := rows.Scan(&user.ID, &user.Name, &user.Surname)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func getKanbanPlaces() ([]PlaceResponse, error) {
	places := []PlaceResponse{}

	rows, err := db.Query(`
        SELECT id, name 
        FROM places 
        ORDER BY name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var place PlaceResponse
		err := rows.Scan(&place.ID, &place.Name)
		if err != nil {
			return nil, err
		}
		places = append(places, place)
	}

	return places, nil
}

func getKanbanMaterials() (map[string][]MaterialResponse, error) {
	materials := make(map[string][]MaterialResponse)

	rows, err := db.Query(`
        SELECT 
            w.id,
            w.name,
            c.name as category,
            w.qty as available
        FROM warehouse w
        JOIN categories c ON w.categorie_id = c.id
        WHERE w.qty > 0
        ORDER BY c.name, w.name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var material MaterialResponse
		err := rows.Scan(&material.ID, &material.Name, &material.Category, &material.Available)
		if err != nil {
			return nil, err
		}

		if materials[material.Category] == nil {
			materials[material.Category] = []MaterialResponse{}
		}
		materials[material.Category] = append(materials[material.Category], material)
	}

	return materials, nil
}

// Обновление статуса задачи
func updateTaskStatus(taskID int, status string) error {
	_, err := db.Exec(`UPDATE tasks SET status = $1 WHERE id = $2`, status, taskID)
	return err
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
