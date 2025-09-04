package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"
)

// Обработчик для Kanban доски
func handleKanban(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadFile("www/kanban.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	t := template.Must(template.New("kanbanPage").Parse(string(buf)))
	t.Execute(w, nil)
}

func handleCompleteKanbanTask(w http.ResponseWriter, r *http.Request) {

	var completeReq struct {
		ID         int             `json:"id"`
		Comments   string          `json:"comments"`
		Materials  []MaterialUsage `json:"materials"`
		FinishDate time.Time       `json:"finish_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&completeReq); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	finishDate := completeReq.FinishDate
	finishDate = finishDate.Add(5 * time.Hour)
	// if err != nil {
	// 	log.Printf("Ошибка парсинга даты: %v", err)
	// 	http.Error(w, "Invalid date format", http.StatusBadRequest)
	// 	return
	// }

	//finishDateUTC := finishDate.UTC()
	log.Printf("Завершение (местрнное): %v", finishDate)

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Обновляем задачу
	result, err := tx.Exec(`
        UPDATE tasks 
        SET status = 'done', 
            finish_date = $1,
            comments = COALESCE($2, comments),
            progress = 100
        WHERE id = $3 AND is_kanban = true
    `, finishDate, completeReq.Comments, completeReq.ID)

	if err != nil {
		log.Printf("Ошибка обновления задачи: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Обновлено строк: %d", rowsAffected)

	// Обрабатываем материалы
	for _, material := range completeReq.Materials {
		log.Printf("Обработка материала: %s - %s (ID: %d) (%d шт.)",
			material.Category, material.Model, material.ModelID, material.Quantity)

		// Используем ID материала
		var currentQty int
		err := tx.QueryRow("SELECT qty FROM warehouse WHERE id = $1 FOR UPDATE", material.ModelID).Scan(&currentQty)

		if err != nil {
			log.Printf("Материал не найден по ID %d: %v", material.ModelID, err)
			continue
		}

		if currentQty < material.Quantity {
			log.Printf("Недостаточно материала ID %d: доступно %d, требуется %d",
				material.ModelID, currentQty, material.Quantity)
			continue
		}

		// Уменьшаем количество
		_, err = tx.Exec("UPDATE warehouse SET qty = qty - $1 WHERE id = $2", material.Quantity, material.ModelID)
		if err != nil {
			log.Printf("Ошибка списания материала: %v", err)
			continue
		}

		// Записываем в историю
		_, err = tx.Exec(
			"INSERT INTO warehouse_history(item_id, task_id, qty, operation_date) VALUES($1, $2, $3, NOW())",
			material.ModelID, completeReq.ID, material.Quantity)

		if err != nil {
			log.Printf("Ошибка записи в историю: %v", err)
		} else {
			log.Printf("Материал успешно списан: ID %d, %s (%d шт.)",
				material.ModelID, material.Model, material.Quantity)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("Ошибка коммита транзакции: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Task completed successfully",
	})
}

// API для обновления статуса задачи
func handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка разбора формы", http.StatusBadRequest)
		return
	}

	taskID, _ := strconv.Atoi(r.FormValue("id"))
	status := r.FormValue("status")

	err := updateTaskStatus(taskID, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Добавляем новые обработчики для Kanban API

func handleGetKanbanTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := getKanbanTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func handleCreateKanbanTask(w http.ResponseWriter, r *http.Request) {
	log.Printf("Получен запрос на создание Kanban задачи")

	var taskReq KanbanTaskRequest
	body, _ := ioutil.ReadAll(r.Body)
	log.Printf("Тело запроса: %s", string(body))

	// Сбрасываем reader чтобы можно было прочитать again
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	if err := json.NewDecoder(r.Body).Decode(&taskReq); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Декодированная задача: %+v", taskReq)

	taskID, err := insertKanbanTask(taskReq)
	if err != nil {
		log.Printf("Ошибка создания задачи: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Задача успешно создана с ID: %d", taskID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"id":      taskID,
		"message": "Task created successfully",
	})
}

func handleUpdateKanbanTaskStatus(w http.ResponseWriter, r *http.Request) {
	var update struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := updateKanbanTaskStatus(update.ID, update.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Task status updated",
	})
}

func handlePauseKanbanTask(w http.ResponseWriter, r *http.Request) {
	var pauseReq PauseRequest
	if err := json.NewDecoder(r.Body).Decode(&pauseReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := pauseKanbanTask(pauseReq.ID, pauseReq.PauseUntil, pauseReq.PauseReason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Task paused successfully",
	})
}

func handleGetKanbanUsers(w http.ResponseWriter, r *http.Request) {
	users, err := getKanbanUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func handleGetKanbanPlaces(w http.ResponseWriter, r *http.Request) {
	places, err := getKanbanPlaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(places)
}

func handleGetKanbanMaterials(w http.ResponseWriter, r *http.Request) {
	materials, err := getKanbanMaterials()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(materials)
}

func handleKanbanPage(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadFile("www/kanban.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	t := template.Must(template.New("kanbanPage").Parse(string(buf)))
	t.Execute(w, nil)
}

func handleGetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := getAllCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func handleGetItems(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category_id")

	var rows *sql.Rows
	var err error

	if categoryID != "" {
		// Фильтрация по категории
		rows, err = db.Query(`
            SELECT id, name, qty, comments 
            FROM warehouse 
            WHERE categorie_id = $1 AND qty > 0
            ORDER BY name
        `, categoryID)
	} else {
		// Все товары (если категория не указана)
		rows, err = db.Query(`
            SELECT id, name, qty, comments 
            FROM warehouse 
            WHERE qty > 0
            ORDER BY name
        `)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name string
		var qty int
		var comments string

		err := rows.Scan(&id, &name, &qty, &comments)
		if err != nil {
			continue
		}

		items = append(items, map[string]interface{}{
			"id":       id,
			"name":     name,
			"qty":      qty,
			"comments": comments,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// Обработчик редактирования товара
func handleEditItem(w http.ResponseWriter, r *http.Request) {
	fmt.Println("edit call")
	if err := r.ParseForm(); err != nil {
		log.Printf("Ошибка ParseForm: %v", err)
		fmt.Printf("Ошибка ParseForm: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Полученные данные: %v", r.Form) // Логируем все данные формы

	id, _ := strconv.Atoi(r.FormValue("id"))
	name := r.FormValue("name")
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	categoryID, _ := strconv.Atoi(r.FormValue("categorie_id"))
	comments := r.FormValue("comments")

	log.Printf("Пытаемся обновить товар ID=%d", id) // Логируем ID
	fmt.Printf("обновляем товар id=%d", id)

	if err := updateItem(id, name, qty, categoryID, comments); err != nil {
		log.Printf("Ошибка updateItem: %v", err) // Логируем ошибку
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Обработчик редактирования категории
func handleEditCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	name := r.FormValue("name")

	if err := updateCategory(id, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleSaveTask(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderErrorPage(w, fmt.Errorf("ошибка разбора формы: %v", err))
		return
	}

	params := r.PostForm
	var id int
	var err error

	// Получаем ID задачи (если есть)
	if idStr := params.Get("id"); idStr != "" {
		id, err = strconv.Atoi(idStr)
		if err != nil {
			renderErrorPage(w, fmt.Errorf("неверный ID задачи: %v", err))
			return
		}
	}

	// Парсим параметры задачи
	namestr := params.Get("user")
	placestr := params.Get("places")
	author := params.Get("description")
	comments := params.Get("comments")
	lenta := params.Has("lenta")
	completedByPetrov := params.Has("completedByPetrov")
	completedByPankov := params.Has("completedByPankov")

	// Парсим даты
	publicationDate, err := time.Parse("2006-01-02T15:04", params.Get("publicationDate"))
	if err != nil {
		renderErrorPage(w, fmt.Errorf("неверный формат даты публикации: %v", err))
		return
	}

	finishDate, err := time.Parse("2006-01-02T15:04", params.Get("finishDate"))
	if err != nil {
		renderErrorPage(w, fmt.Errorf("неверный формат даты завершения: %v", err))
		return
	}
	usedItems := r.Form["used_items"]
	for _, itemIDStr := range usedItems {
		if itemIDStr == "undefined" {
			renderErrorPage(w, fmt.Errorf("не выбран материал"))
			return
		}
		_, err := strconv.Atoi(itemIDStr)
		if err != nil {
			renderErrorPage(w, fmt.Errorf("неверный ID материала: %v", err))
			return
		}
	}
	// Получаем user_id и place_id
	var userID, placeID int
	if id == 0 { // Новая задача
		userID, err = strconv.Atoi(namestr)
		if err != nil {
			renderErrorPage(w, fmt.Errorf("неверный ID пользователя: %v", err))
			return
		}

		placeID, err = strconv.Atoi(placestr)
		if err != nil {
			renderErrorPage(w, fmt.Errorf("неверный ID места: %v", err))
			return
		}
	} else { // Существующая задача
		if namecur, err := strconv.Atoi(namestr); err == nil && strconv.Itoa(namecur) != "0" {
			userID = namecur
		} else {
			userID, err = getUserIdFromName(namestr)
			if err != nil {
				renderErrorPage(w, fmt.Errorf("ошибка получения ID пользователя: %v", err))
				return
			}
		}

		if placecur, err := strconv.Atoi(placestr); err == nil && strconv.Itoa(placecur) != "0" {
			placeID = placecur
		} else {
			placeID, err = getPlaceIdFromName(placestr)
			if err != nil {
				renderErrorPage(w, fmt.Errorf("ошибка получения ID места: %v", err))
				return
			}
		}
	}

	// Сохраняем задачу (создаем или обновляем)
	var taskID int
	if id == 0 {
		taskID, err = insertTask(userID, author, placeID, publicationDate, finishDate,
			comments, lenta, completedByPetrov, completedByPankov)
	} else {
		taskID = id
		_, err = updateTask(taskID, userID, author, placeID, publicationDate, finishDate,
			comments, lenta, completedByPetrov, completedByPankov)
	}

	if err != nil {
		renderErrorPage(w, fmt.Errorf("ошибка сохранения задачи: %v", err))
		return
	}

	// Только после успешного сохранения задачи обрабатываем материалы
	if usedItems := params["used_items"]; len(usedItems) > 0 {
		itemQty, _ := strconv.Atoi(params.Get("item_qty"))

		for _, itemIDStr := range usedItems {
			itemID, err := strconv.Atoi(itemIDStr)
			if err != nil {
				renderErrorPage(w, fmt.Errorf("неверный ID материала: %v", err))
				return
			}

			if err := useItem(itemID, taskID, itemQty); err != nil {
				renderErrorPage(w, fmt.Errorf("ошибка списания материалов: %v", err))
				return
			}
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleSaveUser(w http.ResponseWriter, r *http.Request) {
	var id = 0
	var err error

	r.ParseForm()
	params := r.PostForm
	idStr := params.Get("id")

	if len(idStr) > 0 {
		id, err = strconv.Atoi(idStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}
	}

	name := params.Get("name")
	surname := params.Get("surname")
	jobtitle := params.Get("jobtitle")
	patronymic := params.Get("patronymic")

	birthdayStr := params.Get("birthday")
	var birthday time.Time

	if len(birthdayStr) > 0 {
		birthday, err = time.Parse("02.01.2006", birthdayStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}
	}

	if id == 0 {
		_, err = insertUser(name, surname, patronymic, birthday, jobtitle)
	} else {
		_, err = updateUser(id, name, surname, patronymic, birthday, jobtitle)
	}

	if err != nil {
		renderErrorPage(w, err)
		return
	}

	http.Redirect(w, r, "/", 302)
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры дат из URL
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	var startDate, endDate time.Time
	var err error

	// Если даты не указаны, используем дефолтный диапазон (неделя)
	if startDateStr == "" || endDateStr == "" {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -7) // неделя назад
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}
		// Добавляем 1 день чтобы включить конечную дату
		endDate = endDate.AddDate(0, 0, 1)
	}

	Tasks, err := allTasks(startDate, endDate)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	buf, err := ioutil.ReadFile("www/index.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	var page = IndexPage{
		AllTasks:  Tasks,
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.AddDate(0, 0, -1).Format("2006-01-02"), // вычитаем 1 день обратно
	}

	indexPage := string(buf)
	t := template.Must(template.New("indexPage").Parse(indexPage))
	t.Execute(w, page)
}

func handleWarehouse(w http.ResponseWriter, r *http.Request) {
	categories, err := getAllCategories()
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	items, err := getAllItems()
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	data := struct {
		Categories []Category
		Items      []WarehouseItem
	}{
		Categories: categories,
		Items:      items,
	}

	t, err := template.ParseFiles("www/warehouse.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}
	t.Execute(w, data)
}

func handleGetItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID товара не указан", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID товара", http.StatusBadRequest)
		return
	}

	var item struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Qty        int    `json:"qty"`
		Comments   string `json:"comments"`
		CategoryID int    `json:"categorie_id"`
	}

	err = db.QueryRow(`
        SELECT id, name, qty, comments, categorie_id 
        FROM warehouse 
        WHERE id = $1`, id).Scan(
		&item.ID, &item.Name, &item.Qty, &item.Comments, &item.CategoryID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Товар не найден", http.StatusNotFound)
		} else {
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func handleSaveItem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ошибка разбора формы", http.StatusBadRequest)
		return
	}

	// Получаем все значения формы
	idStr := r.FormValue("id") // Важно: получаем ID из формы
	name := r.FormValue("name")
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	categoryID, _ := strconv.Atoi(r.FormValue("categorie_id"))
	comments := r.FormValue("comments")

	// Валидация данных
	if name == "" || qty < 0 {
		http.Error(w, "Некорректные данные", http.StatusBadRequest)
		return
	}

	// Проверка существования категории
	var categoryExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", categoryID).Scan(&categoryExists)
	if err != nil || !categoryExists {
		http.Error(w, "Категория не существует", http.StatusBadRequest)
		return
	}

	var resultID int
	var message string

	// Если ID передан - обновляем существующий товар
	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Неверный ID товара", http.StatusBadRequest)
			return
		}

		err = updateItem(id, name, qty, categoryID, comments)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resultID = id
		message = "Товар обновлен"
	} else {
		// Если ID не передан - добавляем новый товар
		id, err := insertItem(name, qty, categoryID, comments)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resultID = id
		message = "Товар добавлен"
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": message,
		"id":      resultID,
	})
}

// Обработчик сохранения категории
func handleSaveCategory(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	idStr := r.FormValue("id")
	name := r.FormValue("name")

	var err error
	if idStr == "" {
		_, err = insertCategory(name)
	} else {
		id, _ := strconv.Atoi(idStr)
		err = updateCategory(id, name)
	}

	if err != nil {
		renderErrorPage(w, err)
		return
	}
	http.Redirect(w, r, "/warehouse#categories", http.StatusSeeOther)
}

// Обработчик удаления категории
func handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID категории не указан", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	// Проверяем, есть ли товары в этой категории
	var itemsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM warehouse WHERE categorie_id = $1", id).Scan(&itemsCount)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	if itemsCount > 0 {
		http.Error(w, "Нельзя удалить категорию с товарами", http.StatusBadRequest)
		return
	}

	err = deleteCategory(id)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	http.Redirect(w, r, "/warehouse#categories", http.StatusSeeOther)
}

// Обработчик удаления товара
func handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "ID товара не указан", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	err = deleteItem(id)
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	http.Redirect(w, r, "/warehouse#items", http.StatusSeeOther)
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	Users, err := allUsers()
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	buf, err := ioutil.ReadFile("www/users.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	var page = IndexUserPage{AllUsers: Users}
	usersPage := string(buf)
	t := template.Must(template.New("usersPage").Parse(usersPage))
	t.Execute(w, page)
}

func handleViewTask(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	idStr := params.Get("id")

	userList, err := getAllUsers()
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	placesList, err := getAllPlaces()
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	categories, err := getAllCategories()
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	var currentTask = Task{
		PublicationDate: time.Now(),
		FinishDate:      time.Now().Add(1 * time.Hour),
	}

	if len(idStr) > 0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		currentTask, err = getTask(id)
		if err != nil {
			renderErrorPage(w, err)
			return
		}
	}

	buf, err := ioutil.ReadFile("www/task.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	var page = TaskPage{
		TargetTask:          currentTask,
		UserList:            userList,
		PlacesList:          placesList,
		WarehouseCategories: categories,
	}

	t := template.Must(template.New("TaskPage").Parse(string(buf)))
	err = t.Execute(w, page)
	if err != nil {
		renderErrorPage(w, err)
	}
}

func handleGetWarehouseItems(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.Atoi(r.URL.Query().Get("category_id"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	items, err := getItemsByCategory(categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func handleViewUser(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	idStr := params.Get("id")

	var currentUser = User{}

	if len(idStr) > 0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		currentUser, err = getUser(id)
		if err != nil {
			renderErrorPage(w, err)
			return
		}
	}

	buf, err := ioutil.ReadFile("www/user.html")
	if err != nil {
		renderErrorPage(w, err)
		return
	}

	var page = UserPage{TargetUser: currentUser}
	UserPage := string(buf)
	t := template.Must(template.New("UserPage").Parse(UserPage))
	err = t.Execute(w, page)
	if err != nil {
		renderErrorPage(w, err)
		return
	}
}

func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	idStr := params.Get("id")

	if len(idStr) > 0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		n, err := removeTask(id)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		fmt.Printf("Rows removed: %v\n", n)
	}
	http.Redirect(w, r, "/", 302)
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	idStr := params.Get("id")

	if len(idStr) > 0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		n, err := removeUser(id)
		if err != nil {
			renderErrorPage(w, err)
			return
		}

		fmt.Printf("User Rows removed: %v\n", n)
	}
	http.Redirect(w, r, "/", 302)
}

func renderErrorPage(w http.ResponseWriter, errorMsg error) {
	buf, err := ioutil.ReadFile("www/error.html")
	if err != nil {
		log.Printf("%v\n", err)
		fmt.Fprintf(w, "%v\n", err)
		return
	}

	var page = ErrorPage{ErrorMsg: errorMsg.Error()}
	errorPage := string(buf)
	t := template.Must(template.New("errorPage").Parse(errorPage))
	t.Execute(w, page)
}
