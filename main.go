package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("dheeraj-mcdhee"))
var db *sql.DB

type Card struct {
	ID      int
	Title   string
	Desc    string
	Author  string
	PubDate string
	ImgName string
}

func init() {
	var err error
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   20 * 60,
		HttpOnly: true,
	}
	db, err = sql.Open("sqlite3", "./blog.db")
	if err != nil {
		os.Exit(1)
	}
}
func getUserAndPasswd()(string, string){
	file, err:=os.Open("sensitive.txt")
	if err !=nil {
		fmt.Println("Sensitive info cannot be reached.")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	user:=scanner.Text()
	scanner.Scan()
	pass:=scanner.Text()
	return user,pass
}

func authentication(r *http.Request) bool {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return false
	}
	passVal := session.Values["pass"]
	userVal := session.Values["user"]
	user, ok := userVal.(string)
	if !ok {
		return false
	}
	pass, ok := passVal.(string)
	if !ok {
		return false
	}
	fmt.Println(pass)
	userVerify, passVerify :=getUserAndPasswd()
	if strings.Compare(user, userVerify) == 0 && strings.Compare(pass, passVerify) == 0 {
		return true
	} else {
		return false
	}
}
func blogUploadForm(w http.ResponseWriter, r *http.Request) {
	defer w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	defer w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	defer w.Header().Set("Pragma", "no-cache")
	defer w.Header().Set("X-Accel-Expires", "0")
	if r.Method == "POST" {
		if !authentication(r) {
			http.Redirect(w, r, "/blogAdminLogin/", http.StatusNetworkAuthenticationRequired)
			return
		}
		r.ParseMultipartForm(32 << 50)
		rows, err := getCount()
		if err != nil {
			fmt.Fprintf(w, "Internal server error")
			return
		}
		err = os.MkdirAll("blog/blog_"+strconv.Itoa(rows+1), os.ModePerm)
		if err != nil {
			fmt.Fprintf(w, "Error while uploading file please try again.")
			return
		}
		file, _, _ := r.FormFile("markdown")
		defer file.Close()
		f, _ := os.OpenFile("blog/blog_"+strconv.Itoa(rows+1)+"/body.md", os.O_WRONLY|os.O_CREATE, 0666)
		defer f.Close()
		io.Copy(f, file)
		_, handler, _ := r.FormFile("image0")
		imageLen, _ := strconv.Atoi(r.Form["imageCount"][0])
		for i := 0; i < imageLen; i++ {
			file, handler, err := r.FormFile("image" + strconv.Itoa(i))
			if err != nil {
				return
			}
			f, _ := os.OpenFile("static/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
			io.Copy(f, file)
		}

		stmt, err := db.Prepare("INSERT INTO Blogs (Title,Descri,Author,PubDate,Image) VALUES (?,?,?,?,?)")
		if err != nil {
			fmt.Fprintf(w, "Internal Server Error")
			return
		}
		_, err = stmt.Exec(r.Form["title"][0], r.Form["desc"][0], r.Form["author"][0], r.Form["pubDate"][0], handler.Filename)
		if err != nil {
			fmt.Fprintf(w, "Internal Server Error")
			fmt.Println(err)
			return
		}
		fmt.Fprintf(w, "Upload success")
		return
	}
	http.Redirect(w, r, "/blogAdminLogin/", http.StatusNetworkAuthenticationRequired)
}

func blogAdminLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	if r.Method == "GET" {
		var output []byte
		output, err := ioutil.ReadFile("adminLogin.html")
		if err != nil {
			fmt.Fprintf(w, "Error 404 page could not be found")
			return
		}
		fmt.Fprintf(w, "%s", output)
	} else if r.Method == "POST" {
		r.ParseForm()
		pass := r.Form["password"][0]
		username := r.Form["username"][0]
		userVerify, passVerify :=getUserAndPasswd()
		passErr := bcrypt.CompareHashAndPassword([]byte(passVerify), []byte(pass))
		userErr := bcrypt.CompareHashAndPassword([]byte(userVerify), []byte(username))
		if passErr == nil && userErr == nil {
			session, errSess := store.Get(r, "session-name")
			session.Values["pass"] = passVerify
			session.Values["user"] = userVerify
			if errSess != nil {
				http.Error(w, errSess.Error(), http.StatusInternalServerError)
				return
			}
			session.Save(r, w)
			var output []byte
			output, err := ioutil.ReadFile("adminUpload.html")
			if err != nil {
				fmt.Fprintf(w, "Error 404 page could not be found")
				return
			}
			fmt.Fprintf(w, "%s", output)
		} else {
			var output []byte
			output, err := ioutil.ReadFile("adminLogin.html")
			if err != nil {
				fmt.Fprintf(w, "Error 404 page could not be found")
				return
			}
			fmt.Fprintf(w, "%s", output)
		}
	}

}
func getCount() (int, error) {
	rows, err := db.Query("SELECT count(*) AS count FROM Blogs")
	defer rows.Close()
	if err != nil {
		return 0, err
	}
	rows.Next()
	var len int
	err = rows.Scan(&len)
	if err != nil {
		return 0, err
	}
	return len, nil
}
func blogLister(w http.ResponseWriter, r *http.Request) {
	cards := make([]Card, 0)
	rows, err := db.Query("SELECT * FROM BLOGS;")
	if err != nil {
		fmt.Fprintf(w, "Internal server error")
		return
	}
	defer rows.Close()
	var blogID int
	var title string
	var desc string
	var author string
	var pubDate string
	var imageName string
	layout := "2006-01-02"
	for rows.Next() {
		err := rows.Scan(&blogID, &title, &desc, &author, &pubDate, &imageName)
		if err != nil {
			fmt.Fprintf(w, "Internal server error")
			return
		}
		t, _ := time.Parse(layout, pubDate)

		c := Card{ID: blogID, Title: title, Desc: desc, Author: author, PubDate: t.Format("January 02,2006"), ImgName: imageName}
		cards = append(cards, c)
	}
	t, _ := template.ParseFiles("template/blog.html")
	t.Execute(w, cards)
}

func blogViewer(w http.ResponseWriter, r *http.Request) {
	layout := "2006-01-02"
	id := r.URL.Path[len("/blogs/"):]
	if len(id) == 0 {
		blogLister(w, r)
		return
	}
	iID, err := strconv.Atoi(id)
	if err != nil {
		http.Redirect(w, r, "/blogs/", http.StatusBadRequest)
		return
	}
	stmt, err := db.Prepare("SELECT Title,Author,PubDate,Image FROM Blogs WHERE ID = ?")
	if err != nil {
		http.Redirect(w, r, "/blogs/", http.StatusBadRequest)
		return
	}
	rows, err := stmt.Query(iID)
	var title string
	var author string
	var pubDate string
	var image string
	for rows.Next() {

		err := rows.Scan(&title, &author, &pubDate, &image)
		if err != nil {
			http.Redirect(w, r, "/blogs/", http.StatusNotFound)
			return
		}
		tTime, _ := time.Parse(layout, pubDate)
		p, err := loadPage(iID, title, image, tTime.Format("January 02,2006"), author)
		if err != nil {
			http.Redirect(w, r, "/blogs/", http.StatusNotFound)
			return
		}
		t, _ := template.ParseFiles("template/post.html")
		// fmt.Fprintf(w, "%s", p.Body)
		t.Execute(w, p)

	}

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	page, err := ioutil.ReadFile("index.html")
	if err != nil {
		fmt.Fprintf(w, "%s", "Error 404 page not found")
		return
	}
	fmt.Fprintf(w, "%s", page)
}

func teamHandler(w http.ResponseWriter, r *http.Request) {
	page, err := ioutil.ReadFile("team.html")
	if err != nil {
		fmt.Fprintf(w, "%s", "Error 404 page not found")
		return
	}
	fmt.Fprintf(w, "%s", page)
}

func main() {
	http.HandleFunc("/blogAdminLogin/", blogAdminLogin)
	http.HandleFunc("/adminBlogForm/", blogUploadForm)
	http.HandleFunc("/blogs/", blogViewer)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/teams/", teamHandler)
	http.HandleFunc("/teams", teamHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
