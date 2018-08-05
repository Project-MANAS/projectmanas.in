package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gomarkdown/markdown"
)

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	err := os.MkdirAll("blog/"+p.Title, os.ModePerm)
	if err != nil {
		return err
	}
	filename := "blog/" + p.Title + "/" + p.Title + ".md"
	return ioutil.WriteFile(filename, p.Body, 0600)
}
func loadPage(title string) (*Page, error) {
	filename := "blog/" + title + "/" + title + ".md"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
		fmt.Println("Error: Page could not be found.")
	}
	return &Page{Title: title, Body: markdown.ToHTML(body, nil, nil)}, nil
}
