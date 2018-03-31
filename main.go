package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly"
)

type BitcoinData struct {
	Name              string
	MarketCap         int64
	Price             float64
	Volume_24h        int64
	CirculatingSupply int64
	Change_24h        float64
}

func main() {

	db, err := sql.Open("mysql", "chy:123456@tcp(192.168.56.102:3306)/test")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	f, err := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("open file error:", err)
		os.Exit(2)
	}
	defer f.Close()

	c := colly.NewCollector()

	bitcoinlist := make([]*BitcoinData, 0)

	// Find and visit all links
	c.OnHTML("#currencies > tbody", func(e *colly.HTMLElement) {
		e.DOM.Find("tr").Each(func(i int, s *goquery.Selection) {
			data := &BitcoinData{}
			name, exist := s.Find("td:nth-child(2)").Attr("data-sort")
			if exist {
				data.Name = name
				fmt.Println(data.Name)
				mCapAttr, ex := s.Find("td:nth-child(3)").Attr("data-usd")
				if ex {
					marketCap, err := strconv.ParseFloat(mCapAttr, 64)
					if err == nil {
						data.MarketCap = int64(marketCap)
					} else {
						fmt.Println("parse MarketCap error:", err)
					}
				}

				priceAttr, ex := s.Find("td:nth-child(4)>a").Attr("data-usd")
				if ex {
					price, err := strconv.ParseFloat(priceAttr, 64)
					if err == nil {
						data.Price = price
					} else {
						fmt.Println("parse Price error:", err)
					}
				}

				volumeAttr, ex := s.Find("td:nth-child(5)>a").Attr("data-usd")
				if ex {
					volume, err := strconv.ParseFloat(volumeAttr, 64)
					if err == nil {
						data.Volume_24h = int64(volume)
					} else {
						fmt.Println("parse Volume 24h error:", err)
					}
				}

				csAttr, ex := s.Find("td:nth-child(6)>a").Attr("data-supply")
				if ex {
					supply, err := strconv.ParseFloat(csAttr, 64)
					if err == nil {
						data.CirculatingSupply = int64(supply)
					} else {
						fmt.Println("parse Circulating Supply error:", err)
					}
				}

				changeAttr, ex := s.Find("td:nth-child(7)").Attr("data-percentusd")
				if ex {
					change, err := strconv.ParseFloat(changeAttr, 64)
					if err == nil {
						data.Change_24h = change
					} else {
						fmt.Println("parse Circulating Supply error:", err)
					}
				}

				bitcoinlist = append(bitcoinlist, data)
				cmd := fmt.Sprintf("INSERT INTO bitcoins (name, marketcap, price, volume_24h, circulating_supply, change_24h, time) values ('%s', %d, %f, %d, %d, %f, NOW())",
					data.Name, data.MarketCap, data.Price, data.Volume_24h, data.CirculatingSupply, data.Change_24h)
				_, err := db.Query(cmd)
				if err != nil {
					fmt.Println("insert into database error:", err)
				}
			}
		})
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
		//f.Write(r.Body)
	})

	c.OnScraped(func(_ *colly.Response) {
		bData, _ := json.MarshalIndent(bitcoinlist, "", "  ")
		//bData, _ := json.Marshal(bitcoinlist)
		f.Write(bData)
	})

	//c.Visit("https://coinmarketcap.com/all/views/all/")
	c.Visit("https://coinmarketcap.com/coins/")
}
