package main

import (
	"database/sql"
	"log"
	"net/http"
)

var db *sql.DB

func init() {
	tmpDB, err := sql.Open("postgres", "dbname=kam user=kam password=kamkam host=localhost sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	db = tmpDB
}

func main() {
	// 1. Сначала статические файлы
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("www/assets"))))

	// 2. API endpoints (должны быть перед основными роутами)
	http.HandleFunc("/api/warehouse/categories", handleGetCategories)
	http.HandleFunc("/api/warehouse/items", handleGetItems)
	http.HandleFunc("/getItem", handleGetItem)

	// 3. Основные роуты
	http.HandleFunc("/warehouse/", handleWarehouse) // Обратите внимание на слеш в конце
	http.HandleFunc("/users", handleListUsers)
	http.HandleFunc("/task.html", handleViewTask)
	http.HandleFunc("/user.html", handleViewUser)

	// 4. Action handlers
	http.HandleFunc("/editItem", handleEditItem)
	http.HandleFunc("/editCategory", handleEditCategory)
	http.HandleFunc("/saveCategory", handleSaveCategory)
	http.HandleFunc("/deleteCategory", handleDeleteCategory)
	http.HandleFunc("/saveItem", handleSaveItem)
	http.HandleFunc("/deleteItem", handleDeleteItem)
	http.HandleFunc("/save", handleSaveTask)
	http.HandleFunc("/saveuser", handleSaveUser)
	http.HandleFunc("/delete", handleDeleteTask)
	http.HandleFunc("/deleteuser", handleDeleteUser)
	// Kanban API endpoints
	http.HandleFunc("/kanban", handleKanbanPage)
	http.HandleFunc("/api/kanban/tasks", handleGetKanbanTasks)
	http.HandleFunc("/api/kanban/tasks/create", handleCreateKanbanTask)
	http.HandleFunc("/api/kanban/tasks/update-status", handleUpdateKanbanTaskStatus)
	http.HandleFunc("/api/kanban/tasks/pause", handlePauseKanbanTask)
	http.HandleFunc("/api/kanban/users", handleGetKanbanUsers)
	http.HandleFunc("/api/kanban/places", handleGetKanbanPlaces)
	http.HandleFunc("/api/kanban/materials", handleGetKanbanMaterials)

	// 5. Дефолтный роут (должен быть ПОСЛЕДНИМ)
	http.HandleFunc("/", handleListTasks)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
