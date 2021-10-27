package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	StockSymbol string `json:"stockSymbol" db:"stock_symbol"`
}

func Index(c *gin.Context) {
	c.IndentedJSON(200, gin.H{"message": "gopher hello"})
}
func ReplaceSQL(old, searchPattern string) string {
	tmpCount := strings.Count(old, searchPattern)
	for m := 1; m <= tmpCount; m++ {
		old = strings.Replace(old, searchPattern, "$"+strconv.Itoa(m), 1)
	}
	return old
}
func FetchAndParseStockData(c *gin.Context) {
	stockToFetch := c.Query("stockToFetch")
	sendRequest(stockToFetch, c)
}

func sendRequest(stockToFetch string, c *gin.Context) {
	var stock datahelpers.Stock
	sendString := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%s&apikey=%s", stockToFetch, os.Getenv("STOCK_API_KEY"))
	resp, err := http.Get(sendString)
	if err != nil || resp.StatusCode != 200 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "Stock fetch failed"})
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "Something Went Wrong"})
		return
	}
	var responseinterface datahelpers.ErrorResponse
	err = json.Unmarshal(body, &responseinterface)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError})
		return
	}
	if responseinterface.ErrorMessage != "" {
		c.IndentedJSON(http.StatusNoContent, gin.H{"code": http.StatusNoContent, "message": "Try Something else"})
		return
	}
	err = json.Unmarshal(body, &stock)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError})
		return
	}
	remappedData := datahelpers.ReMap(stock)
	pg_error, reg_error := dbinsert(remappedData)
	if pg_error != "" || reg_error != nil {
		if pg_error == pgerrcode.UniqueViolation {
			c.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"code": http.StatusUnprocessableEntity, "message": "Stock Already Exists in database"})
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "Something went wrong..."})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "Successful fetch & insert"})
}

func dbinsert(stock datahelpers.StockNew) (pq.ErrorCode, error) {
	query := fmt.Sprintf("INSERT INTO %s(%s) VALUES($1) RETURNING id;", "STOCK_INFO_TABLE", "STOCK_SYMBOL")
	var stockId int32
	db := databasetools.ConnectToDatabase()
	_, err := db.Exec(db.Rebind(query), stock.NewMeta.Symbol)
	if err, ok := err.(*pq.Error); ok {
		if err.Code == pgerrcode.UniqueViolation {
			return err.Code, nil
		}
	}
	errors := db.QueryRowx("SELECT id FROM STOCK_INFO_TABLE WHERE stock_symbol=$1", stock.NewMeta.Symbol).Scan(&stockId)
	if errors != nil {
		return "", errors
	}
	vals := []interface{}{}
	secondQuery := fmt.Sprintf("INSERT INTO %s(%s,%s,%s,%s,%s,%s) VALUES ", "STOCK_VALUE_TABLE", "stock_date", "stock_symbol_id", "stock_open", "stock_high", "stock_low", "stock_close")
	for key := range stock.NewTime {
		secondQuery += "(?, ?, ?, ?, ?, ?),"
		//fmt.Println(key, remappedData.NewTime[key].Open, remappedData.NewTime[key].High, remappedData.NewTime[key].Low, remappedData.NewTime[key].Close)
		vals = append(vals, key, stockId, stock.NewTime[key].Open, stock.NewTime[key].High, stock.NewTime[key].Low, stock.NewTime[key].Close)
	}
	secondQuery = strings.TrimSuffix(secondQuery, ",")
	secondQuery = ReplaceSQL(secondQuery, "?")
	stmt, _ := db.Prepare(secondQuery)
	res, err := stmt.Exec(vals...)
	if err != nil {
		return "", err
	}
	fmt.Println(res.RowsAffected())
	return "", nil
}

func TruncTable(c *gin.Context) {
	db := databasetools.ConnectToDatabase()
	db.Exec("TRUNCATE TABLE STOCK_INFO_TABLE CASCADE")
}

func GetStockByName(c *gin.Context) {
	fmt.Println("I'm here")
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
	}
	response, err := getRestOfDataForStock(stockResponse)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "Something went wrong"})
		return
	}
	c.IndentedJSON(200, gin.H{"info": stockResponse, "stockValues": response})

}

func getRestOfDataForStock(stockResponse StockResponse) ([]datahelpers.StockAttributes, error) {
	db := databasetools.ConnectToDatabase()
	rows, err := db.Queryx(fetchStockInfo, stockResponse.Id)
	if err != nil {
		return nil, err
	}
	stockList := []datahelpers.StockAttributes{}
	defer rows.Close()
	for rows.Next() {
		var stockData datahelpers.StockAttributes
		err := rows.StructScan(&stockData)
		if err != nil {
			return nil, err
		}
		stockList = append(stockList, stockData)
	}
	return stockList, nil
}

func GetAllStocks(c *gin.Context) {
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

func HelloAdmin(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": fmt.Sprintf("hello %s", os.Getenv("USERADMIN"))})
}
