package main

import (
	"fmt"
)

// ========== Обработчики для категорий ==========
// Проверка перед удалением категории
func canDeleteCategory(id int) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM warehouse WHERE categorie_id = $1", id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func getAllCategories() ([]Category, error) {
	categories := []Category{}
	rows, err := db.Query("SELECT id, name FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Category
		err := rows.Scan(&c.ID, &c.Name)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}
func checkMaterialAvailability(category, model string, quantity int) (bool, error) {
	var available int
	err := db.QueryRow(`
        SELECT w.qty 
        FROM warehouse w
        JOIN categories c ON w.categorie_id = c.id
        WHERE c.name = $1 AND w.name = $2
    `, category, model).Scan(&available)

	if err != nil {
		return false, err
	}

	return available >= quantity, nil
}

func insertCategory(name string) (int, error) {
	var id int
	err := db.QueryRow("INSERT INTO categories(name) VALUES($1) RETURNING id", name).Scan(&id)
	return id, err
}

func updateCategory(id int, name string) error {
	_, err := db.Exec("UPDATE categories SET name=$1 WHERE id=$2", name, id)
	return err
}

func deleteCategory(id int) error {
	_, err := db.Exec("DELETE FROM categories WHERE id=$1", id)
	return err
}

func getItemsByCategory(categoryID int) ([]WarehouseItem, error) {
	items := []WarehouseItem{}
	rows, err := db.Query(`
        SELECT 
            id, 
            name, 
            COALESCE(qty, 0) as qty,
            comments
        FROM warehouse 
        WHERE categorie_id = $1 AND qty > 0
        ORDER BY name`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса товаров: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item WarehouseItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Qty, &item.Comments); err != nil {
			return nil, fmt.Errorf("ошибка сканирования товара: %v", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func getAllItems() ([]WarehouseItem, error) {
	items := []WarehouseItem{}
	query := `
		SELECT w.id, w.name, w.qty, w.categorie_id, c.name as category_name, w.comments 
		FROM warehouse w 
		JOIN categories c ON w.categorie_id = c.id
		ORDER BY c.name, w.name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item WarehouseItem
		err := rows.Scan(&item.ID, &item.Name, &item.Qty, &item.CategoryID, &item.CategoryName, &item.Comments)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func insertItem(name string, qty int, categoryID int, comments string) (int, error) {
	// Проверяем существование категории
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)", categoryID).Scan(&exists)
	if err != nil || !exists {
		return 0, fmt.Errorf("категория с ID %d не существует", categoryID)
	}

	// Добавляем товар
	var id int
	err = db.QueryRow(
		"INSERT INTO warehouse(name, qty, categorie_id, comments) VALUES($1, $2, $3, $4) RETURNING id",
		name, qty, categoryID, comments,
	).Scan(&id)

	return id, err
}

func updateItem(id int, name string, qty int, categoryID int, comments string) error {
	fmt.Println(id, name, qty, categoryID, comments)
	_, err := db.Exec(
		"UPDATE warehouse SET name=$1, qty=$2, categorie_id=$3, comments=$4 WHERE id=$5",
		name, qty, categoryID, comments, id)
	return err
}

func deleteItem(id int) error {
	_, err := db.Exec("DELETE FROM warehouse WHERE id=$1", id)
	return err
}

func useItem(itemID, taskID, qty int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Проверяем доступное количество
	var currentQty int
	err = tx.QueryRow("SELECT qty FROM warehouse WHERE id = $1 FOR UPDATE", itemID).Scan(&currentQty)
	if err != nil {
		return err
	}

	if currentQty < qty {
		return fmt.Errorf("недостаточно товара на складе")
	}

	// Уменьшаем количество
	_, err = tx.Exec("UPDATE warehouse SET qty = qty - $1 WHERE id = $2", qty, itemID)
	if err != nil {
		return err
	}

	// Записываем в историю (нужно создать таблицу warehouse_history)
	_, err = tx.Exec(
		"INSERT INTO warehouse_history(item_id, task_id, qty, operation_date) VALUES($1, $2, $3, NOW())",
		itemID, taskID, qty)
	if err != nil {
		return err
	}

	return tx.Commit()
}
