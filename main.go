package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("something-very-secret"))
var db *sql.DB

type Card struct {
	ID      int
	Title   string
	Desc    string
	ImgName string
}

func init() {
	var err error
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   20 * 60 * 1000,
		HttpOnly: true,
	}
	db, err = sql.Open("sqlite3", "./blog.db")
	if err != nil {
		os.Exit(1)
	}
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
	userVerify := os.Getenv("USER")
	passVerify := os.Getenv("PASS")
	if strings.Compare(user, userVerify) == 0 && strings.Compare(pass, passVerify) == 0 {
		return true
	} else {
		return false
	}
}
func blogUploadForm(w http.ResponseWriter, r *http.Request) {
	// replacer := strings.NewReplacer(",", "&comma;")
	if r.Method == "POST" {
		if !authentication(r) {
			var output []byte
			output, err := ioutil.ReadFile("adminLogin.html")
			if err != nil {
				fmt.Fprintf(w, "Error 404 page could not be found")
				return
			}
			fmt.Fprintf(w, "%s", output)
			return
		}
		r.ParseMultipartForm(32 << 50)
		// csvFile, _ := os.OpenFile("blogs.csv", os.O_RDWR|os.O_APPEND, 0660)
		// defer csvFile.Close()
		// csvReader := csv.NewReader(bufio.NewReader(csvFile))
		// csvData, err := csvReader.ReadAll()
		// if err != nil {
		// 	fmt.Fprintf(w, "Error while uploading file please try again.")
		// }
		// rows := len(csvData)
		rows, err := getCount()
		if err != nil {
			fmt.Fprintf(w, "Internal server error")
			return
		}
		err = os.MkdirAll("blog/blog_"+strconv.Itoa(rows), os.ModePerm)
		if err != nil {
			fmt.Fprintf(w, "Error while uploading file please try again.")
			return
		}
		file, _, _ := r.FormFile("markdown")
		defer file.Close()
		f, _ := os.OpenFile("blog/blog_"+strconv.Itoa(rows)+"/body.md", os.O_WRONLY|os.O_CREATE, 0666)
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

		stmt, err := db.Prepare("INSERT INTO Blogs (Title,Descri,Image) VALUES (?,?,?)")
		if err != nil {
			fmt.Fprintf(w, "Internal Server Error")
			fmt.Println("here1")
			return
		}
		_, err = stmt.Exec(r.Form["title"][0], r.Form["desc"][0], handler.Filename)
		if err != nil {
			fmt.Fprintf(w, "Internal Server Error")
			fmt.Println(err)
			return
		}
		// vals := []string{strconv.Itoa(rows + 1), replacer.Replace(r.Form["title"][0]), replacer.Replace(r.Form["desc"][0]), replacer.Replace(handler.Filename)}
		// csvData = append(csvData, vals)
		// writer := csv.NewWriter(csvFile)
		// defer writer.Flush()
		// for _, val := range csvData {
		// 	_ = writer.Write(val)
		// }
		fmt.Fprintf(w, "Upload success")
		return
	}
	fmt.Fprintf(w, "Error uploading")
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
		userVerify := os.Getenv("USER")
		passVerify := os.Getenv("PASS")
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
	var imageName string
	for rows.Next() {
		err := rows.Scan(&blogID, &title, &desc, &imageName)
		if err != nil {
			fmt.Fprintf(w, "Internal server error")
			return
		}
		c := Card{ID: blogID, Title: title, Desc: desc, ImgName: imageName}
		cards = append(cards, c)
	}
	// len, err := getCount()
	// fmt.Println(len, err)
	// if err != nil {
	// 	fmt.Fprintf(w, "Internal server error")
	// }
	t, _ := template.ParseFiles("template/blog.html")
	// fmt.Fprintf(w, "%s", strconv.Itoa(len))
	t.Execute(w, cards)
}

func blogViewer(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/blogs/"):]
	if len(title) == 0 {
		blogLister(w, r)
		return
	}
	p, err := loadPage(title)
	if err != nil {
		fmt.Fprintf(w, "<h1>%s</h1>", "Error 404 unable to find the blog")
	} else {
		fmt.Fprintf(w, "%s", p.Body)
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
	http.HandleFunc("/blogs", blogViewer)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/teams/", teamHandler)
	http.HandleFunc("/teams", teamHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
