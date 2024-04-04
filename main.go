package main

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// type status

type ItemStatus int

const (
	ItemStatusDoing ItemStatus = iota
	ItemStatusDone
	ItemStatusDeleted
)

var allItemsStatus = [3]string{"Doing", "Done", "Deleted"}

func (item *ItemStatus) String() string {
	return allItemsStatus[*item]
}

func parseStrItemStatus(s string) (ItemStatus, error) {
	for i := range allItemsStatus {
		if allItemsStatus[i] == s {
			return ItemStatus(i), nil
		}
	}
	return ItemStatus(0), errors.New("invalid status string")
}

func (item *ItemStatus) Scan(value interface{}) error {
	bytes, ok := value.([]byte)

	if !ok {
		return errors.New(fmt.Sprintf("fail to scan data from sql: %s", value))
	}

	v, err := parseStrItemStatus(string(bytes))

	if err != nil {
		return errors.New(fmt.Sprintf("fail to scan data from sql: %s", value))
	}

	*item = v

	return nil

}

func (item *ItemStatus) Value() (driver.Value, error) {

	if item == nil {

		return nil, nil
	}

	return item.String(), nil
}

func (item *ItemStatus) MarshalJSON() ([]byte, error) {

	if item == nil {
		return nil, nil
	}

	return []byte(fmt.Sprintf("\"%s\"", item.String())), nil
}

func (item *ItemStatus) UnMarshalJSON(data []byte) error {

	str := strings.ReplaceAll(string(data), "\"", "") // "Doing"

	itemValue, err := parseStrItemStatus(str)

	if err != nil {
		return nil
	}

	*item = itemValue

	return nil
}

// type todo
type ToDoItem struct {
	Id        int         `json:"id" gorm:"column:id;"`
	Title     string      `json:"title" gorm:"column:title;"`
	Status    *ItemStatus `json:"status" gorm:"column:status;"`
	CreatedAt *time.Time  `json:"created_at" gorm:"column:created_at;"`
	UpdatedAt *time.Time  `json:"updated_at" gorm:"column:updated_at;"`
}

//
func (ToDoItem) TableName() string {
	return "todo_items"
}

type Paging struct {
	Page  int   `json:"page" form:"page"`
	Limit int   `json:"Limit" form:"limit"`
	Total int64 `json:"total" form:"-"`
}

func (p *Paging) Process() {
	if p.Page <= 0 {
		p.Page = 1
	}

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 10
	}
}

// ham chinh

func main() {
	dsn := "root:123456@tcp(127.0.0.1:3306)/todo_db?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalln("Cannot connect to MySQL:", err)
	}

	log.Println("Connected:", db)

	router := gin.Default()

	v1 := router.Group("/v1")

	{
		v1.POST("/items", createItem(db))      // create item
		v1.GET("/items", getListOfitems(db))   // get all items
		v1.GET("/items/:id", getItemByID(db))  // get one itemby ID
		v1.PATCH("/items/:id", updateItem(db)) // update item
		v1.DELETE("items/:id", deleteItem(db)) // delete item
	}

	router.Run()
}

func createItem(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var dataItem ToDoItem

		if err := ctx.ShouldBind(&dataItem); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		// title trim all spaces
		dataItem.Title = strings.TrimSpace(dataItem.Title)

		if dataItem.Title == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": "title cannot be blank"})
			return
		}

		// status default = doing
		// dataItem.Status = "Doing"

		if err := db.Create(&dataItem).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"data": dataItem})
	}

}

func getListOfitems(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var paging Paging

		if err := ctx.ShouldBind(&paging); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		paging.Process()

		var result []ToDoItem

		db = db.Where("status <> ?", "Deleted")

		if err := db.Table(ToDoItem{}.TableName()).Count(&paging.Total).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		if err := db.Order("id desc").
			Offset((paging.Page - 1) * paging.Limit).
			Limit(paging.Limit).
			Find(&result).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"data": result, "paging": paging})
	}
}

func getItemByID(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var dataItem ToDoItem

		id, err := strconv.Atoi(ctx.Param("id"))

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		dataItem.Id = id // cach 1

		// db.Where("id = ?" , id).First(&dataItem) cach 2
		if err := db.First(&dataItem).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"data": dataItem})
	}
}

func updateItem(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var dataItem ToDoItem

		id, err := strconv.Atoi(ctx.Param("id"))

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		log.Print("id", id)
		log.Print("err", err)

		if err := ctx.ShouldBind(&dataItem); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		dataItem.Id = id

		if err := db.Updates(&dataItem).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"data": true})
	}
}

func deleteItem(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var dataItem ToDoItem

		id, err := strconv.Atoi(ctx.Param("id"))

		log.Print("id", id)
		log.Print("err", err)

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		dataItem.Id = id

		if err := db.Delete(&dataItem).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"data": true})
	}
}
