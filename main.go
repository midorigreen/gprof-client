package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	ui "github.com/gizak/termui"
	"github.com/midorigreen/gprof-client/prof"
	"github.com/midorigreen/gprof-client/prof/cpu"
	"github.com/midorigreen/gprof-client/prof/disk"
	"github.com/midorigreen/gprof-client/prof/file"
)

type Config struct {
	URL     string  `toml:"url"`
	GQParam GQParam `toml:"gq_param"`
}

type GQParam struct {
	DiskPath string `toml:"disk_path"`
	FilePath string `toml:"file_path"`
	Num      int    `toml:"num"`
}

var widgetMap map[string]prof.ProfWidget = map[string]prof.ProfWidget{}

func initWidget() {
	widgetMap["cpu"] = cpu.CreateWidget()
	widgetMap["file"] = file.CreateWidget()
	widgetMap["disk"] = disk.CreateWidget()
}

func run() error {
	var config Config
	_, err := toml.DecodeFile("./config.toml", &config)
	if err != nil {
		return err
	}

	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	prof, err := req(config)
	if err != nil {
		ui.StopLoop()
	}

	initWidget()

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(12, 0, widgetMap["cpu"].Create(prof)...),
		),
		ui.NewRow(
			ui.NewCol(6, 0, widgetMap["file"].Create(prof)...),
			ui.NewCol(6, 0, widgetMap["disk"].Create(prof)...),
		),
	)
	ui.Render(ui.Body)

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(e ui.Event) {
		ui.Body.Align()
		prof, err := req(config)
		if err != nil {
			ui.StopLoop()
		}
		for _, v := range widgetMap {
			v.Update(prof)
		}

		ui.Render(ui.Body)
		time.Sleep(1 * time.Second)
	})

	ui.Loop()
	return nil
}

func req(config Config) (prof.Prof, error) {
	query, err := createQuery(config.GQParam)
	if err != nil {
		return prof.Prof{}, err
	}

	postBody := fmt.Sprintf(`{"query": %s}`, strconv.QuoteToASCII(query))
	resp, err := http.Post(config.URL, "application/json", strings.NewReader(postBody))
	if err != nil {
		return prof.Prof{}, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var prof prof.Prof
	err = decoder.Decode(&prof)
	if err != nil && err != io.EOF {
		return prof, err
	}
	return prof, nil
}

func createQuery(param GQParam) (string, error) {
	tmpl, err := template.ParseFiles("./template/template.json")
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	tmpl.Execute(&b, param)

	return b.String(), nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
