package main

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tenntenn/connpass"
)

func main() {
	var morningEvents []map[string]string
	var lunchtimeEvents []map[string]string

	var baseParams []connpass.Param
	baseParams = append(baseParams, connpass.Count(100))
	baseParams = append(baseParams, connpass.Order(connpass.OrderByDate))
	// 2週間分取得
	now := time.Now()
	for i := 0; i < 14; i++ {
		d := now.AddDate(0, 0, i)
		baseParams = append(baseParams, connpass.YearMonthDay(d.Year(), d.Month(), d.Day()))
	}

	cli := connpass.NewClient()
	ctx := context.Background()

	// 最後まで結果を取得する
	// response例 {"results_start": 801, "results_returned": 57, "results_available": 857 ...
	start := 1
	for {
		params, err := connpass.SearchParam(append(baseParams, connpass.Start(start))...)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("search params:" + params.Encode())

		r, err := cli.Search(ctx, params)
		if err != nil {
			log.Fatal(err)
		}

		// 結果が0なら終了
		if len(r.Events) == 0 {
			break
		}

		for _, e := range r.Events {
			if (e.StartedAt.Day() == e.EndedAt.Day()) &&
				(e.StartedAt.Hour() >= 6 && e.EndedAt.Hour() <= 10) {
				morningEvents = append(morningEvents, convertInfo(e))
			}

			if (e.StartedAt.Day() == e.EndedAt.Day()) &&
				(e.StartedAt.Hour() >= 11 && e.EndedAt.Hour() <= 14) {
				lunchtimeEvents = append(lunchtimeEvents, convertInfo(e))
			}
		}

		// 最後まで取得できていれば次を実行せずに終了
		if (r.Start + r.Returned - 1) >= r.Available {
			break
		}

		start += 100

		// https://connpass.com/robots.txt
		//   Crawl-delay: 5
		time.Sleep(time.Second * 5)
	}

	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		log.Fatal(err)
	}
	outputPath := "./index.html"
	f, _ := os.Create(outputPath)
	w := bufio.NewWriter(f)

	if err := t.Execute(w, struct {
		LastUpdate      string
		LunchtimeEvents []map[string]string
		MorningEvents   []map[string]string
	}{
		LastUpdate:      time.Now().Format("2006/01/02 15:04"),
		LunchtimeEvents: lunchtimeEvents,
		MorningEvents:   morningEvents,
	}); err != nil {
		log.Printf("failed to execute template: %v", err)
	}
	w.Flush()
	if err != nil {
		log.Fatal(err)
	}
}

func convertInfo(e *connpass.Event) map[string]string {
	weekdayJa := strings.NewReplacer(
		"Sun", "日",
		"Mon", "月",
		"Tue", "火",
		"Wed", "水",
		"Thu", "木",
		"Fri", "金",
		"Sat", "土",
	)

	return map[string]string{
		"Date":       weekdayJa.Replace(e.StartedAt.Format("1/2(Mon)")),
		"Time":       e.StartedAt.Format("15:04") + " - " + e.EndedAt.Format("15:04"),
		"Title":      e.Title,
		"TittleLink": e.URL,
		"Group":      e.Series.Title,
		"GroupLink":  e.Series.URL,
	}
}
