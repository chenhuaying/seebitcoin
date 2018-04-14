package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly"
	"github.com/spf13/viper"
)

type BitcoinData struct {
	Name              string
	Symbol            string
	MarketCap         int64
	Price             float64
	Volume_24h        int64
	CirculatingSupply int64
	Change_24h        float64
}

const (
	DEFAULT_USER            = "chy"
	DEFAULT_PASSWD          = "123456"
	DEFAULT_IP              = "192.168.56.102"
	DEFAULT_PORT            = "3306"
	DEFAULT_DATABASE        = "test"
	DEFAULT_TABLE           = "bitcoins"
	DEFAULT_TIMEOUT         = 30
	DEFAULT_MARKETCAP_FLUSH = 1200

	CONF_KEY_DBUSER          = "dbUser"
	CONF_KEY_DBPASSWD        = "dbPasswd"
	CONF_KEY_DBIP            = "dbIP"
	CONF_KEY_DBPORT          = "dbPort"
	CONF_KEY_DATABASE        = "database"
	CONF_KEY_TIMEOUT         = "timeout"
	CONF_KEY_MARKETCAP_FLUSH = "marketcap_flush"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.seebitcoin")
	viper.AddConfigPath("/etc/seebitcoin")

	viper.SetDefault(CONF_KEY_DBUSER, DEFAULT_USER)
	viper.SetDefault(CONF_KEY_DBPASSWD, DEFAULT_PASSWD)
	viper.SetDefault(CONF_KEY_DBIP, DEFAULT_IP)
	viper.SetDefault(CONF_KEY_DBPORT, DEFAULT_PORT)
	viper.SetDefault(CONF_KEY_DATABASE, DEFAULT_DATABASE)
	viper.SetDefault(CONF_KEY_TIMEOUT, DEFAULT_TIMEOUT)
	viper.SetDefault(CONF_KEY_MARKETCAP_FLUSH, DEFAULT_MARKETCAP_FLUSH)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("read config error: ", err)
	}

	dbUser := viper.GetString(CONF_KEY_DBUSER)
	dbPasswd := viper.GetString(CONF_KEY_DBPASSWD)
	dbAddr := viper.GetString(CONF_KEY_DBIP)
	dbPort := viper.GetString(CONF_KEY_DBPORT)
	database := viper.GetString(CONF_KEY_DATABASE)
	timeout := viper.GetInt(CONF_KEY_TIMEOUT)
	mcapFlush := viper.GetInt64(CONF_KEY_MARKETCAP_FLUSH)

	dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPasswd, dbAddr, dbPort, database)

	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	infos := GetBitcoinsInfo(db)

	f, err := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("open file error:", err)
		os.Exit(2)
	}
	defer f.Close()

	c := colly.NewCollector()
	c.SetRequestTimeout(time.Duration(timeout) * time.Second)
	c.WithTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 30 * time.Second,
	})

	bitcoinlist := make([]*BitcoinData, 0)
	timestamp := time.Now().Unix()

	c.OnHTML("#currencies-all > tbody", func(e *colly.HTMLElement) {
		e.DOM.Find("tr").Each(func(i int, s *goquery.Selection) {
			data := &BitcoinData{}
			name, exist := s.Find("td:nth-child(2)").Attr("data-sort")
			if exist {
				data.Name = name
				fmt.Println(data.Name)

				data.Symbol = s.Find("td:nth-child(3)").Text()

				mCapAttr, ex := s.Find("td:nth-child(4)").Attr("data-usd")
				if ex {
					marketCap, err := strconv.ParseFloat(mCapAttr, 64)
					if err == nil {
						data.MarketCap = int64(marketCap)
					} else {
						fmt.Println("parse MarketCap error:", err)
					}
				}

				priceAttr, ex := s.Find("td:nth-child(5)>a").Attr("data-usd")
				if ex {
					price, err := strconv.ParseFloat(priceAttr, 64)
					if err == nil {
						data.Price = price
					} else {
						fmt.Println("parse Price error:", err)
					}
				}

				volumeAttr, ex := s.Find("td:nth-child(7)>a").Attr("data-usd")
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

				changeAttr, ex := s.Find("td:nth-child(9)").Attr("data-percentusd")
				if ex {
					change, err := strconv.ParseFloat(changeAttr, 64)
					if err == nil {
						data.Change_24h = change
					} else {
						fmt.Println("parse Circulating Supply error:", err)
					}
				}

				bitcoinlist = append(bitcoinlist, data)

				id, ok := infos[data.Symbol+"_"+data.Name]
				if !ok {
					lid, err := AddInfo(data.Name, data.Symbol, db)
					if err != nil {
						fmt.Println("add new coin error:", err)
						id = -1
					} else {
						id = lid
					}
				}

				if id >= 0 {
					t := timestamp % mcapFlush
					if t > mcapFlush-31 || t < 30 {
						_, err = AddMarketCap(id, data.MarketCap, data.CirculatingSupply, data.Volume_24h, data.Change_24h, db)
						if err != nil {
							fmt.Printf("add %s, id(%d), MarketCap(%d), CirculatingSupply(%d), Volume_24h(%d), Change_24h(%f) error: %s\n",
								data.Name, id, data.MarketCap, data.CirculatingSupply, data.Volume_24h, data.Change_24h, err)
						}
					}

					_, err = AddPrice(id, data.Price, db)
					if err != nil {
						fmt.Println("add %s, id(%d), Price(%f), error: %s\n", data.Name, id, data.Price)
					}
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

	//err = c.Visit("https://coinmarketcap.com/coins/views/all/")
	err = c.Visit("https://coinmarketcap.com/all/views/all/")
	if err != nil {
		fmt.Println("vist coins all error:", err)
	}
}
