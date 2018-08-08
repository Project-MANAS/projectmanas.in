package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/gomarkdown/markdown"
)

type Page struct {
	Title   string
	Body    string
	PubDate string
	Author  string
	Image   string
}

func (p *Page) save() error {
	err := os.MkdirAll("blog/"+p.Title, os.ModePerm)
	if err != nil {
		return err
	}
	filename := "blog/" + p.Title + "/" + p.Title + ".md"
	return ioutil.WriteFile(filename, []byte(p.Body), 0600)
}
func loadPage(ID int, title, image, date, author string) (*Page, error) {
	filename := "blog/blog_" + strconv.Itoa(ID) + "/body.md"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Error: Page could not be found.")
		return nil, err
	}
	return &Page{Title: title, Body: string(markdown.ToHTML(body, nil, nil)), Image: image, PubDate: date, Author: author}, nil
}
