package main

import (
	"context"
	"fmt"
	"github.com/gocolly/colly/v2"
	"strconv"
	"regexp"
	"github.com/influxdata/influxdb-client-go/v2"
	"time"
	"flag"
)


type realEstate struct {
	id string
	price int
	size int
	plotSize int
	location string
	rooms int

}

func saveSnapshoot(influxToken string, data []realEstate) {

	client := influxdb2.NewClient("http://localhost:8086", influxToken)
    writeAPI := client.WriteAPIBlocking("admin", "berci")

	now := time.Now()

	for _, v := range data {
		p := influxdb2.NewPointWithMeasurement("ingatlan").
        AddTag("id", v.id).
        AddTag("location", v.location).
        AddTag("size", strconv.FormatInt(int64(v.size), 10)).
        AddTag("plotSize", strconv.FormatInt(int64(v.plotSize), 10)).
        AddTag("rooms", strconv.FormatInt(int64(v.rooms), 10)).
        AddField("price", v.price).
        SetTime(now)
    	writeAPI.WritePoint(context.Background(), p)
	}
}

func scrape() []realEstate {
	c := colly.NewCollector()
	// c := colly.NewCollector(colly.MaxDepth(4),)

	data := []realEstate{}

	
	c.OnHTML(".listing", func(e *colly.HTMLElement) {

		id := e.Attr("data-id")

		priceFloat, _ := strconv.ParseFloat(getValueByRegex("(\\d+).*", e.ChildText(".price")), 32)

		price := int(priceFloat * 1000000)
		size, _ := strconv.Atoi(getValueByRegex("(\\d+).*", e.ChildText(".listing__data--area-size")))
		plotSize, _ := strconv.Atoi(getValueByRegex("(\\d+).*", e.ChildText(".listing__data--plot-size")))
		rooms, _ := strconv.Atoi(getValueByRegex("(\\d+).*", e.ChildText(".listing__data--room-count")))
		location := e.ChildText(".listing__address")

		realEstateInfo := realEstate{id: id, price: price, size: size, plotSize: plotSize, rooms: rooms, location: location}

		data = append(data, realEstateInfo)
		
	})

	c.OnHTML(".pagination__inner :last-child > a", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting %v\n", r.URL)
	})

	c.Visit("https://ingatlan.com/lista/elado+haz+uj-epitesu+pest-megye")

	return data
}


func scrapeAndSave(influxToken string) {
	fmt.Println("start scrape")
	data := scrape()
	fmt.Println("scrape done")
	// fmt.Println(data)

	fmt.Println("saveing")
	saveSnapshoot(influxToken, data)
	fmt.Println("save done")
}

func main() {


	influxTokenPtr := flag.String("influx_token", "", "influx  token")

	flag.Parse()

	influxToken := *influxTokenPtr

	if influxToken == "" {
		panic("you have to provide the influx token")
	}


	for range time.Tick(time.Minute * 5) {
		go scrapeAndSave(influxToken)
	}
}

func getValueByRegex(regex string, data string) string{
	re := regexp.MustCompile(regex)
	m := re.FindStringSubmatch(data)
	return m[1]
}
