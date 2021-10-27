package datahelpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type MetaData struct {
	Information   string `json:"1. Information"`
	Symbol        string `json:"2. Symbol"`
	LastRefreshed string `json:"3. Last Refreshed"`
	OutputSize    string `json:"4. Output Size"`
	TimeZone      string `json:"5. Time Zone"`
}

type Stats struct {
	Open  string `json:"1. open"`
	High  string `json:"2. high"`
	Low   string `json:"3. low"`
	Close string `json:"4. close"`
}
type StockAttributes struct {
	StockSymbolId int    `json:"StockSymbolId" db:"stock_symbol_id"`
	Open          string `json:"Open" db:"stock_open"`
	High          string `json:"High" db:"stock_high"`
	Low           string `json:"Low" db:"stock_low"`
	Close         string `json:"Close" db:"stock_close"`
}

type Stock struct {
	Metadata   MetaData         `json:"Meta Data"`
	TimeSeries map[string]Stats `json:"Time Series (Daily)"`
}

type StockNew struct {
	NewMeta struct {
		Information   string `json:"Information"`
		Symbol        string `json:"Symbol"`
		LastRefreshed string `json:"LastRefreshed"`
		OutputSize    string `json:"OutputSize"`
		TimeZone      string `json:"TimeZone"`
	} `json:"MetaData"`
	NewTime map[string]Stats `json:"NewTime"`
}

type StockOutPut struct {
	Id     string                     `json:"id"`
	Symbol string                     `json:"stocksymbol"`
	TimeZs map[string]StockAttributes `json:"Attributes I guess"`
}

type ErrorResponse struct {
	ErrorMessage string `json:"Error Message"`
}

func ReMap(stock Stock) StockNew {
	newStock := StockNew{
		NewMeta: struct {
			Information   string "json:\"Information\""
			Symbol        string "json:\"Symbol\""
			LastRefreshed string "json:\"LastRefreshed\""
			OutputSize    string "json:\"OutputSize\""
			TimeZone      string "json:\"TimeZone\""
		}(stock.Metadata),
		NewTime: stock.TimeSeries,
	}

	return newStock

}

func withInterFace() {
	jsonFile, err := os.Open("data/stock_material.json")
	//newStock := Stock{} // but new
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()
	result := make(map[string]interface{})
	bytevalue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal([]byte(bytevalue), &result)
}
