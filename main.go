package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"reloading-api/databasetools"
	"reloading-api/datahelpers"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

const (
	sqlFetchStockByName = "SELECT id, stock_symbol FROM STOCK_INFO_TABLE WHERE stock_symbol=$1"
	sqlFetchAllStocks   = "SELECT id, stock_symbol FROM STOCK_INFO_TABLE"
	fetchStockInfo      = "SELECT stock_symbol_id, stock_open, stock_high, stock_low, stock_close FROM STOCK_VALUE_TABLE where stock_symbol_id = $1"
	emptyValueMessage   = "Nothing found"
	genericErrorMessage = "Something went wrong"
	sqlPostQuery        = "INSERT INTO %s(%s,%s) VALUES($1, $2)"
	successFullPost     = "Success"
)

type StockResponse struct {
	Id          int32  `json:"id" db:"id"`
	StockSymbol string `json:"userName" db:"stock_symbol"`
}

func index(c *gin.Context) {
	c.IndentedJSON(200, gin.H{"message": "gopher hello"})
}
func ReplaceSQL(old, searchPattern string) string {
	tmpCount := strings.Count(old, searchPattern)
	for m := 1; m <= tmpCount; m++ {
		old = strings.Replace(old, searchPattern, "$"+strconv.Itoa(m), 1)
	}
	return old
}
func fetchAndParseStockData(c *gin.Context) {
	var stock datahelpers.Stock
	err := c.BindJSON(&stock)
	if err != nil {
		fmt.Println(err)
	}
	remappedData := datahelpers.ReMap(stock)
	pg_error := dbinsert(remappedData)
	if pg_error {
		c.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"code": http.StatusUnprocessableEntity, "message": "Stock already fetched"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": "Stock inserted and can be queried"})
}
func dbinsert(stock datahelpers.StockNew) bool {
	query := fmt.Sprintf("INSERT INTO %s(%s) VALUES($1) RETURNING id;", "STOCK_INFO_TABLE", "STOCK_SYMBOL")
	db := databasetools.ConnectToDatabase()
	_, err := db.Exec(db.Rebind(query), stock.NewMeta.Symbol)
	if err, ok := err.(*pq.Error); ok {
		if err.Code == pgerrcode.UniqueViolation {
			return true
		}
	}
	var stockId int32
	errors := db.QueryRowx("SELECT id FROM STOCK_INFO_TABLE WHERE stock_symbol=$1", stock.NewMeta.Symbol).Scan(&stockId)
	if err != nil {
		fmt.Println(errors.Error())
	}
	vals := []interface{}{}
	fmt.Println(stockId)
	secondQuery := fmt.Sprintf("INSERT INTO %s(%s,%s,%s,%s,%s,%s) VALUES ", "STOCK_VALUE_TABLE", "stock_date", "stock_symbol_id", "stock_open", "stock_high", "stock_low", "stock_close")
	for key := range stock.NewTime {
		secondQuery += "(?, ?, ?, ?, ?, ?),"
		//fmt.Println(key, remappedData.NewTime[key].Open, remappedData.NewTime[key].High, remappedData.NewTime[key].Low, remappedData.NewTime[key].Close)
		vals = append(vals, key, stockId, stock.NewTime[key].Open, stock.NewTime[key].High, stock.NewTime[key].Low, stock.NewTime[key].Close)
	}
	fmt.Println(vals)
	secondQuery = strings.TrimSuffix(secondQuery, ",")
	secondQuery = ReplaceSQL(secondQuery, "?")
	stmt, _ := db.Prepare(secondQuery)
	res, err := stmt.Exec(vals...)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(res)
	return false
}

func getRestOfDataForStock(stockResponse StockResponse) []datahelpers.StockAttributes {
	db := databasetools.ConnectToDatabase()
	rows, err := db.Queryx(fetchStockInfo, stockResponse.Id)
	if err != nil {
		fmt.Println(err)
	}
	stockList := []datahelpers.StockAttributes{}
	defer rows.Close()
	for rows.Next() {
		var stockData datahelpers.StockAttributes
		err := rows.StructScan(&stockData)
		if err != nil {
			fmt.Println(err)
		}
		stockList = append(stockList, stockData)
	}
	fmt.Println(len(stockList))
	return stockList
}

func truncTable(c *gin.Context) {
	db := databasetools.ConnectToDatabase()
	db.Exec("TRUNCATE TABLE STOCK_INFO_TABLE CASCADE")
}

func getStockByName(c *gin.Context) {
	stockSymbol, ok := c.GetQuery("stockSymbol")
	if !ok {
		c.IndentedJSON(401, gin.H{"message": "give me a param"})
	}
	db := databasetools.ConnectToDatabase()
	var stockResponse StockResponse
	err := db.QueryRowx(sqlFetchStockByName, stockSymbol).StructScan(&stockResponse)
	defer db.Close()
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(400, gin.H{"message": emptyValueMessage})
			return
		}
		c.IndentedJSON(500, gin.H{"message": genericErrorMessage})
		return
	} else {
		response := getRestOfDataForStock(stockResponse)
		c.IndentedJSON(200, gin.H{"info": stockResponse, "stockValues": response})
	}
}

func getAllStocks(c *gin.Context) {
	db := databasetools.ConnectToDatabase()
	rows, err := db.Queryx(sqlFetchAllStocks)
	if err != nil {
		if err == sql.ErrNoRows {
			c.IndentedJSON(400, gin.H{"message": emptyValueMessage})
			return
		}
		c.IndentedJSON(500, gin.H{"message": genericErrorMessage})
		return
	}
	defer rows.Close()
	stockList := []StockResponse{}
	for rows.Next() {
		var fetchedStock StockResponse
		err := rows.StructScan(&fetchedStock)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": genericErrorMessage})
			return
		}
		stockList = append(stockList, fetchedStock)
	}
	c.IndentedJSON(http.StatusOK, gin.H{"code": http.StatusOK, "stockList": stockList})
}

func helloAdmin(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": fmt.Sprintf("hello %s", os.Getenv("USERADMIN"))})
}

func main() {
	router := gin.Default()
	router.POST("/", index)
	getRoutergroup := router.Group("/get", gin.BasicAuth(gin.Accounts{
		os.Getenv("GETGROUPUSER"): os.Getenv("GETGROUPWD"),
	}))
	adminGroup := router.Group("/admin", gin.BasicAuth(gin.Accounts{
		os.Getenv("USERADMIN"): os.Getenv("USERADMINPWD"),
	}))
	getRoutergroup.GET("/by-stocksymbol", getStockByName)
	getRoutergroup.GET("/allstocks", getAllStocks)
	adminGroup.GET("", helloAdmin)
	adminGroup.POST("/insertStock", fetchAndParseStockData)
	router.GET("/trunc", truncTable)
	router.Run(":8080")

}
