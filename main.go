package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	page     = make(map[string][]byte)
	session  = make(map[string]string)
	titles   = make([]TitleInfo, 0)
	articles = make(map[string]Article)
)

type TitleInfo struct {
	Title  string
	Author string
}

/**
文章信息
*/
type Article struct {
	Title   string  //标题
	Name    string  //作者
	Content string  //正文
	Reply   []Reply //评论
}

/**
评论信息
*/
type Reply struct {
	FromName string
	Content  string
	ToName   string
	IsFirst  bool
}

func init() {
	loadHtml("index", "./views/index.html")
	loadHtml("login", "./views/login.html")
	loadHtml("register", "./views/register.html")
	loadHtml("details", "./views/details.html")
	loadHtml("upload", "./views/upload.html")
}

func loadHtml(key, path string) {
	info, err := readFile(path)
	if err != nil {
		return
	}
	page[key] = info
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func main() {
	http.HandleFunc("/toLogin", toLogin)   //跳转到登录页
	http.HandleFunc("/login", login)       //进行登录操作
	http.HandleFunc("/reg", toRegister)    //跳转到注册页面
	http.HandleFunc("/register", register) //进行注册
	http.HandleFunc("/logout", logout)     //注销当前登录
	http.HandleFunc("/home", home)         //首页
	http.HandleFunc("/details", details)   //文章详情页
	http.HandleFunc("/toUpload", toUpload) //跳转到添加文章页面
	http.HandleFunc("/upload", upload)     //添加文章
	http.HandleFunc("/reply", reply)       //评论
	http.HandleFunc("/respond", respond)   //跳转到回复窗口
	server := http.Server{
		Addr: ":8000",
	}
	server.ListenAndServe()
}

func respond(w http.ResponseWriter, r *http.Request) {
	if !isLogin(r) {
		http.Redirect(w, r, "/toLogin", http.StatusFound)
		return
	}
	token, _ := r.Cookie("token")
	name := session[token.Value]
	toname := r.FormValue("toName")
	title := r.FormValue("title")
	tmpl := template.Must(template.ParseFiles("./views/respond.html"))
	tmpl.Execute(w, map[string]interface{}{
		"name":   name,
		"toName": toname,
		"title":  title,
	})
}

func reply(w http.ResponseWriter, r *http.Request) {
	if !isLogin(r) {
		http.Redirect(w, r, "/toLogin", http.StatusFound)
		return
	}
	token, _ := r.Cookie("token")
	name := session[token.Value]
	toname := r.FormValue("toName")
	title := r.FormValue("title")
	content := r.FormValue("content")
	article := articles[title]
	var rep Reply
	rep.Content = content
	rep.FromName = name
	rep.ToName = toname
	rep.IsFirst = toname == ""
	article.Reply = append(article.Reply, rep)
	articles[title] = article
	http.Redirect(w, r, "/details?title="+title, http.StatusFound)
}

func toUpload(w http.ResponseWriter, r *http.Request) {
	if !isLogin(r) {
		http.Redirect(w, r, "/toLogin", http.StatusFound)
		return
	}
	fmt.Fprintf(w, "%s", page["upload"])
}
func upload(w http.ResponseWriter, r *http.Request) {
	if !isLogin(r) {
		http.Redirect(w, r, "/toLogin", http.StatusFound)
		return
	}
	cookie, _ := r.Cookie("token")
	name := session[cookie.Value]
	title := r.FormValue("title")
	content := r.FormValue("content")
	//增加文章信息
	var art Article
	art.Content = content
	art.Name = name
	art.Title = title
	fmt.Println("新增文章：", art)
	articles[title] = art
	//首页展示信息
	var titleinfo TitleInfo
	titleinfo.Author = name
	titleinfo.Title = title
	titles = append(titles, titleinfo)
	http.Redirect(w, r, "/home", http.StatusFound)
}

func details(w http.ResponseWriter, r *http.Request) {
	if !isLogin(r) {
		http.Redirect(w, r, "/toLogin", http.StatusFound)
		return
	}
	token, _ := r.Cookie("token")
	name := session[token.Value]
	tmpl := template.Must(template.ParseFiles("./views/details.html"))
	title := r.FormValue("title")
	article := articles[title]
	flag := len(article.Reply) == 0
	tmpl.Execute(w, map[string]interface{}{
		"name":    name,
		"article": article,
		"isReply": !flag,
	})
}

func home(w http.ResponseWriter, r *http.Request) {
	if !isLogin(r) {
		http.Redirect(w, r, "/toLogin", http.StatusFound)
		return
	}
	tmpl := template.Must(template.ParseFiles("./views/index.html"))
	token, _ := r.Cookie("token")
	name := session[token.Value]
	flag := len(titles) == 0
	tmpl.Execute(w, map[string]interface{}{"name": name, "titles": titles, "isNothing": flag})
}

func register(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	pwd := r.FormValue("pwd")
	if len(name) == 0 && len(pwd) == 0 {
		fmt.Fprintf(w, "%s", page["register"])
		return
	}
	f, err := os.OpenFile("./data/user.txt", os.O_WRONLY, 0644)
	if err != nil {
		w.Write([]byte("注册失败！"))
	}
	n, _ := f.Seek(0, os.SEEK_END)
	f.WriteAt([]byte(name+","+pwd+"\n"), n)
	w.Write([]byte("注册成功！"))
}

func toRegister(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", page["register"])
}

func toLogin(w http.ResponseWriter, r *http.Request) {
	if isLogin(r) {
		http.Redirect(w, r, "/home", http.StatusFound)
		return
	}
	fmt.Fprintf(w, "%s", page["login"])
}

func login(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	pwd := r.FormValue("pwd")
	if len(name) == 0 || len(pwd) == 0 {
		fmt.Fprintf(w, "%s", page["login"])
		return
	}
	userfile, err := os.Open("./data/user.txt")
	defer userfile.Close()
	if err != nil {
		w.Write([]byte("出错了！"))
		return
	}
	var flag bool
	read := bufio.NewReader(userfile)
	for {
		buf, _, err1 := read.ReadLine()
		if err1 != nil {
			if err1 == io.EOF {
				flag = false
				break
			}
		}
		res := strings.Split(string(buf), ",")
		if name == res[0] && pwd == res[1] {
			flag = true
			break
		}
	}
	if flag {
		addCookie(w, name)
		fmt.Println(session)
		http.Redirect(w, r, "/home", http.StatusFound)
		return
	} else {
		fmt.Fprintf(w, "%s", page["login"])
		return
	}
}

func addCookie(w http.ResponseWriter, name string) {
	token := getUUID()
	cookie := http.Cookie{Name: "token", Value: token, MaxAge: 86400}
	http.SetCookie(w, &cookie)
	session[token] = name
}

func getUUID() string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	bytes := make([]byte, 16)
	for i := 0; i < 16; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func isLogin(r *http.Request) bool {
	cookie, err := r.Cookie("token")
	if err != nil {
		return false
	}
	name := session[cookie.Value]
	if name == "" {
		return false
	}
	return true
}

//注销登录
func logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return
	}
	delete(session, cookie.Value)
	fmt.Println(session)
	http.Redirect(w, r, "/toLogin", http.StatusFound)
}
