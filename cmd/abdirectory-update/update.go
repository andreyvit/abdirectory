package main

import (
	"bytes"
	"context"
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"flag"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/oauth2/google"
	"gopkg.in/Iwark/spreadsheet.v2"
)

//go:embed template.html
var htmlTemplate string

var secretFile string
var outputFile string
var spreadsheetID string
var daemonInterval time.Duration
var daemon bool

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	flag.StringVar(&secretFile, "secret", "client_secret.json", "secret file")
	flag.StringVar(&outputFile, "o", "", "output file")
	flag.StringVar(&spreadsheetID, "id", "1G1waMZJ3qdyv3rco7tzxHN8emPJnY73uC-ApZor3X-o", "spreadsheet id")
	flag.DurationVar(&daemonInterval, "interval", time.Minute, "daemon repeat interval")
	flag.BoolVar(&daemon, "d", false, "daemon mode")
	flag.Parse()

	if daemon {
		log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		interceptShutdownSignals(cancel)
		runDaemon(ctx)
	} else {
		err := run(context.Background())
		if err != nil {
			log.Fatalf("** %v", err)
		}
		log.Printf("OK")
	}
}

func runDaemon(ctx context.Context) {
	for ctx.Err() == nil {
		err := run(ctx)
		if err != nil {
			log.Printf("** Failed: %v", err)
		} else {
			log.Printf("Finished.")
		}

		select {
		case <-ctx.Done():
			break
		case <-time.After(daemonInterval):
			break
		}
	}
}

func run(ctx context.Context) error {
	clientSecret, err := ioutil.ReadFile(secretFile)
	if err != nil {
		return err
	}

	conf, err := google.JWTConfigFromJSON(clientSecret, spreadsheet.Scope)
	if err != nil {
		panic(err)
	}

	client := conf.Client(ctx)

	log.Printf("Loading spreadsheet...")

	service := spreadsheet.NewServiceWithClient(client)
	spread, err := service.FetchSpreadsheet(spreadsheetID)
	if err != nil {
		return fmt.Errorf("FetchSpreadsheet: %w", err)
	}

	data := &Data{
		Title:    "AB.Brand июнь 2021",
		EditLink: "https://docs.google.com/spreadsheets/d/1G1waMZJ3qdyv3rco7tzxHN8emPJnY73uC-ApZor3X-o/edit?usp=sharing",
		People:   loadPeople(&spread),
		Chats:    loadChats(&spread),
	}

	tmpl := template.New("")
	tmpl.Funcs(template.FuncMap{
		"formatText": formatText,
	})
	tmpl.Parse(htmlTemplate)
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("template: %w", err)
	}

	if outputFile == "" {
		fmt.Fprintln(os.Stdout, buf.String())
	} else {
		err := os.WriteFile(outputFile, buf.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("WriteFile: %w", err)
		}
	}

	return nil
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func loadPeople(spread *spreadsheet.Spreadsheet) []*Person {
	data := loadSheet(spread, 0)
	result := make([]*Person, 0, len(data))
	for _, row := range data {
		p := &Person{
			Name:      row["Имя Фамилия"],
			Niche:     row["Коротко о вашей нише"],
			Age:       row["Возраст"],
			City:      row["Город сейчас"],
			Instagram: row["Ник в Instagram"],
			Telegram:  row["Ник в Telegram"],
			PointA:    row["Точка A"],
			PointB:    row["Точка B"],
			Bio:       row["О себе"],
		}
		p.ID = md5str(strings.ToLower(p.Name))
		p.InstagramLink = makeInstagramLink(p.Instagram)
		p.TelegramLink = makeTelegramLink(p.Telegram)
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func loadChats(spread *spreadsheet.Spreadsheet) []*Chat {
	data := loadSheet(spread, 1)
	result := make([]*Chat, 0, len(data))
	for _, row := range data {
		p := &Chat{
			Name:     row["Группа"],
			Link:     row["Ссылка"],
			Comments: row["Комментарии"],
		}
		result = append(result, p)
	}
	return result
}

func loadSheet(spread *spreadsheet.Spreadsheet, sheetIndex int) []map[string]string {
	sheet, _ := spread.SheetByIndex(uint(sheetIndex))
	if sheet == nil {
		return nil
	}

	var headers []string
	for _, cell := range sheet.Rows[0] {
		headers = append(headers, cell.Value)
	}

	result := make([]map[string]string, 0, len(sheet.Rows)-1)
	for _, row := range sheet.Rows[1:] {
		m := make(map[string]string, len(row))
		var nonEmpty bool
		for i, cell := range row {
			if i < len(headers) && headers[i] != "" {
				v := strings.TrimSpace(cell.Value)
				m[headers[i]] = v
				if v != "" {
					nonEmpty = true
				}
			}
		}
		if nonEmpty {
			result = append(result, m)
		}
	}

	// log.Printf("sheet %d = %v", sheetIndex)

	return result
}

type Data struct {
	Title    string
	EditLink string
	People   []*Person
	Chats    []*Chat
}

type Chat struct {
	Name     string
	Link     string
	Comments string
}

type Person struct {
	ID            string
	Name          string
	Niche         string
	Age           string
	City          string
	Instagram     string
	InstagramLink string
	Telegram      string
	TelegramLink  string
	PointA        string
	PointB        string
	Bio           string
}

func (p *Person) NameExtras() string {
	var extras []string
	if p.Age != "" {
		extras = append(extras, p.Age)
	}
	if p.City != "" {
		extras = append(extras, p.City)
	}
	return strings.Join(extras, ", ")
}

func formatText(text string) template.HTML {
	var buf strings.Builder

	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		var lines []string
		for _, line := range strings.Split(para, "\n") {
			line = strings.TrimSpace(line)
			lines = append(lines, line)
		}

		buf.WriteString("<p>\n")
		for i, line := range lines {
			if i > 0 {
				buf.WriteString("<br>\n")
			}
			buf.WriteString(html.EscapeString(line))
			buf.WriteString("\n")
		}
		buf.WriteString("</p>\n")
	}

	return template.HTML(buf.String())
}

func md5str(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func makeInstagramLink(s string) string {
	if s == "" {
		return ""
	}

	if !strings.Contains(s, "/") && !strings.Contains(s, "?") {
		s = strings.TrimPrefix(s, "@")
		return "https://instagram.com/" + s + "/"
	}

	return s
}

func makeTelegramLink(s string) string {
	if s == "" {
		return ""
	}

	if !strings.Contains(s, "/") && !strings.Contains(s, "?") {
		s = strings.TrimPrefix(s, "@")
		return "https://t.me/" + s
	}

	return s
}

func interceptShutdownSignals(shutdown func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGHUP)
	go func() {
		<-c
		signal.Reset()
		log.Println("shutting down, interrupt again to force quit")
		shutdown()
	}()
}
