package main

import (
	"fmt"
	"strconv"
	"time"
)

// IndexPage represents the content of the index page, available on "/"
// The index page shows a list of all Tasks stored on db
type IndexPage struct {
	AllTasks  []Task
	StartDate string // Добавляем
	EndDate   string // Добавляем
}

type IndexCategoriesPage struct {
	Categories []Categorie
}

type Categorie struct {
	ID   int
	Name string
}

type CategoriePage struct {
	TargetCategorie Categorie
}

type IndexUserPage struct {
	AllUsers []User
}

type IndexItemPage struct {
	AllItems []Item
}

// TaskPage represents the content of the Task page, available on "/Task.html"
// The Task page shows info about a given Task
type TaskPage struct {
	TargetTask          Task
	UserList            []UserList
	PlacesList          []PlacesList
	WarehouseCategories []Category
}

type UserPage struct {
	TargetUser User
}

type WarehouseCategory struct {
	ID   int
	Name string
}
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WarehouseItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Qty          int    `json:"qty"`
	Comments     string `json:"comments,omitempty"`
	CategoryID   int    `json:"category_id"`
	CategoryName string `json:"category_name,omitempty"`
}

type ItemPage struct {
	TargetItem Item
}

type IndexUserList struct {
	UserList []UserList
}

type IndexPlacesList struct {
	PlacesList []PlacesList
}

// Структуры для API ответов
type UserResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

type PlaceResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type MaterialResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Available int    `json:"available"`
}

type PauseRequest struct {
	ID          int       `json:"id"`
	PauseUntil  time.Time `json:"pause_until"`
	PauseReason string    `json:"pause_reason"`
}

// Структура для запроса создания Kanban задачи
type KanbanTaskRequest struct {
	UserID      int             `json:"user_id"`
	PlaceID     int             `json:"place_id"`
	Description string          `json:"description"`
	Deadline    string          `json:"deadline"` // Меняем time.Time на string
	Comments    string          `json:"comments"`
	Urgent      bool            `json:"urgent"`
	Status      string          `json:"status"`
	Materials   []MaterialUsage `json:"materials,omitempty"`
}

// Структура для ответа с Kanban задачами
type KanbanTask struct {
	ID          int             `json:"id"`
	Title       string          `json:"title"`
	User        string          `json:"user"`
	UserID      int             `json:"user_id"`
	Description string          `json:"description"`
	StartDate   time.Time       `json:"start_date"`
	Deadline    time.Time       `json:"deadline"`
	Status      string          `json:"status"`
	Urgent      bool            `json:"urgent"`
	Progress    int             `json:"progress"`
	Paused      bool            `json:"paused"`
	PauseUntil  time.Time       `json:"pause_until,omitempty"`
	PauseReason string          `json:"pause_reason,omitempty"`
	Comments    string          `json:"comments,omitempty"`
	Materials   []MaterialUsage `json:"materials,omitempty"`
}

type MaterialUsage struct {
	Category string `json:"category"`
	Model    string `json:"model"`
	ModelID  int    `json:"model_id"` // Добавьте это поле
	Quantity int    `json:"quantity"`
}

// Task represents a Task object
type Task struct {
	ID                int
	User              string
	Userid            int
	TaskDescription   string
	UserList          UserList
	PublicationDate   time.Time
	FinishDate        time.Time
	Duration          time.Duration
	Lenta             bool
	Places            string
	Placeid           int
	PlacesList        PlacesList
	Comments          string
	CompletedByPetrov bool
	CompletedByPankov bool
	IsKanban          bool
}

type User struct {
	ID         int
	Name       string
	Surname    string
	Patronymic string
	Birthday   time.Time
	Jobtitle   string
}

type Item struct {
	ID           int
	Name         string
	Qty          int
	Comments     string
	Categorie_id int
}

type UserList struct {
	ID      int
	Name    string
	Surname string
}

type PlacesList struct {
	ID   int
	Name string
}
type Equipment struct {
	ID     int
	UserID int

	// Основная информация о ПК
	PCName              string
	ConnectionType      string
	WindowsVersion      string
	OfficeVersion       string
	RestorePointEnabled bool
	HasZoom             bool
	HasSynapse          bool
	PCAutoUpdate        bool   `json:"pc_auto_update"`
	PCYear              string `json:"pc_year"`
	IPAddress           string `json:"ip_address"`
	FreeSpaceC          string `json:"free_space_c"`
	FreeSpaceD          string `json:"free_space_d"`
	Monitors            string `json:"monitors"`
	Printers            string `json:"printers"`
	KESVersion          string `json:"kes_version"`
	KSCVersion          string `json:"ksc_version"`
	SynapseVersion      string `json:"synapse_version"`
	ZoomVersion         string `json:"zoom_version"`

	// Телефон
	PhoneNumber string
	PhoneModel  string

	// Характеристики ПК
	CPU         string
	CPUYear     int
	RAM         string
	Motherboard string
	GPU         string
	Storage1    string
	Storage2    string
	Storage3    string

	CreatedAt time.Time
	UpdatedAt time.Time

	// Периферия
	Monitor       string
	SecondMonitor string

	// Оборудование со склада - ДОБАВИТЬ
	Keyboard           string `json:"keyboard"`
	Mouse              string `json:"mouse"`
	KeyboardMouseCombo string `json:"keyboard_mouse_combo"`
	Webcam             string `json:"webcam"`
	Speakers           string `json:"speakers"`
	Headset            string `json:"headset"`
	UPS                string `json:"ups"`
	Cartridges         string `json:"cartridges"` // вместо PrinterCartridges

	// Ноутбук
	// Ноутбук - ДОБАВИТЬ
	LaptopAutoUpdate     bool   `json:"laptop_auto_update"`
	LaptopModel          string `json:"laptop_model"`
	LaptopConnectionType string `json:"laptop_connection_type"`
	LaptopIPAddress      string `json:"laptop_ip_address"`
	LaptopYear           string `json:"laptop_year"`
	LaptopFreeSpaceC     string `json:"laptop_free_space_c"`
	LaptopFreeSpaceD     string `json:"laptop_free_space_d"`
	LaptopMonitors       string `json:"laptop_monitors"`
	LaptopPrinters       string `json:"laptop_printers"`
	LaptopKESVersion     string `json:"laptop_kes_version"`
	LaptopKSCVersion     string `json:"laptop_ksc_version"`
	LaptopSynapseVersion string `json:"laptop_synapse_version"`
	LaptopZoomVersion    string `json:"laptop_zoom_version"`
	FortiClientVersion   string `json:"forti_client_version"`
	LaptopName           string
	LaptopIssueYear      int
	LaptopWindowsVersion string
	LaptopOfficeVersion  string
	LaptopVPNVersion     string
	LaptopHasSkype       bool
	LaptopHasSynapse     bool
	LaptopHasZoom        bool
	LaptopLastCheckDate  time.Time

	// Другое оборудование
	Printer           string
	PrinterCartridges string
	PrinterAddress    string

	OtherEquipment string

	// Примечания
	Notes    string
	Problems string
}

// PublicationDateStr returns a sanitized Publication Date in the format YYYY-MM-DD
func (b Task) PublicationDateStr() string {
	return b.PublicationDate.Format("2006-01-02T15:04")
}

func (b Task) PublicationDateView() string {
	return b.PublicationDate.Format("02.01.2006 15:04")
}

func (b User) BirthdayDateStr() string {
	return b.Birthday.Format("02.01.2006")
}

func (b Task) FinishDateStr() string {
	return b.FinishDate.Format("2006-01-02T15:04")
}

func (b Task) FinishDateView() string {
	return b.FinishDate.Format("02.01.2006 15:04")
}

func (b Task) LentaTaskView() string {
	var answer string
	if b.Lenta == true {
		answer = "да"
	} else if b.Lenta == false {
		answer = "нет"
	}
	return answer
}

func (b Task) CompletedByPetrovView() string {
	var answer string
	if b.CompletedByPetrov == true {
		answer = "да"
	} else if b.CompletedByPetrov == false {
		answer = "нет"
	}
	return answer
}

func (b Task) CompletedByPankovView() string {
	var answer string
	if b.CompletedByPankov == true {
		answer = "да"
	} else if b.CompletedByPankov == false {
		answer = "нет"
	}
	return answer
}

func (b Task) IntervalStr() string {
	d1 := b.Duration
	d1 = d1 / time.Hour
	d2 := b.Duration
	hstr := d1.String()[:len(d1.String())-2]
	hint, _ := strconv.Atoi(hstr)
	minutes := d2 / time.Minute
	mstr := minutes.String()[:len(minutes.String())-2]
	mint, _ := strconv.Atoi(mstr)

	if mint >= 60 {
		mint = mint - (60 * hint)
	}
	fmt.Println(hint, " hours ", mint, "minutes")
	var split string
	if hint == 0 {
		split = (strconv.Itoa(mint) + " мин.")
	}
	if mint == 0 {
		split = (strconv.Itoa(hint) + " ч. ")
	}
	if hint > 0 && mint > 0 {
		split = (strconv.Itoa(hint) + " ч. " + strconv.Itoa(mint) + " мин.")
	}

	return split
}

// ErrorPage represents shows an error message, available on "/Task.html"
type ErrorPage struct {
	ErrorMsg string
}
