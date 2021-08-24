package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"goWeb/model"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	// Initialize connection constants.
	HOST     = "localhost"
	DATABASE = "cat_db"
	USER     = "postgres"
	PASSWORD = os.Getenv("PASSWORD")
)

var (
	db                *sql.DB
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     os.Getenv("ClientID"),
		ClientSecret: os.Getenv("ClientSecret"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
	randomState = "random"
)

func init() {
	err := initDB()
	checkError(err)

}

func initDB() (err error) {

	// Initialize connection string.
	var connectionString string = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", HOST, USER, PASSWORD, DATABASE)

	// Initialize connection object.
	db, err = sql.Open("postgres", connectionString)
	if err != nil {
		return
	}

	err = db.Ping()
	if err != nil {
		return
	}
	fmt.Println("Successfully created connection to database")
	return
}

func checkError(err error) {
	if err != nil {
		log.Fatal("err=", err)
	}
}
func main() {
	r := gin.Default()
	r.LoadHTMLGlob("view/*")
	r.Static("/asset", "./asset")

	store := sessions.NewCookieStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	r.GET("/ping", HelloWorld)
	r.GET("/", GetIndex)
	r.GET("/checkSession", checkSession)
	r.GET("/loginGoogle", handleGoogleLogin)
	r.GET("/callback", callbackHandler)
	r.GET("/postAdoptArticle", postAdoptArticleHandler)

	r.GET("/allAdoptArticles", allAdoptArticlesHandler)

	r.POST("/createUser", createUserHandler)
	r.POST("/modifyUser", modifyUserHandler)
	r.POST("/verifyUser", verifyUserHandler)
	r.POST("/logout", loggedInHandler)

	r.POST("/createAdoptArticle", createAdoptArticleHandler)

	r.GET("/User", queryUserHandler)
	r.GET("/login", loginHandler)
	r.GET("/checkLoggedIn", checkLoggedInHandler)
	r.Run() // listen and serve (for windows "http://localhost:8080")
}

func createAdoptArticleHandler(c *gin.Context) {

	session := sessions.Default(c)
	//sessionID := session.Get("loginuser")
	userID := session.Get("userId")
	fmt.Println("userid=", userID)
	if userID == nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   "請先登入",
		})
		return
	}

	var id int
	err := db.QueryRow(`INSERT INTO cat_adopt_posts(city, area,cat_name,cat_age,cat_personality,cat_story,title,author_id) VALUES ($1, $2, $3,$4,$5,$6,$7,$8) returning id`, c.PostForm("city"), c.PostForm("area"), c.PostForm("catName"), c.PostForm("catAge"), c.PostForm("catPersonality"), c.PostForm("catStory"), c.PostForm("title"), userID.(int)).Scan(&id)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}
	file, err := c.FormFile("catPicture")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}
	c.SaveUploadedFile(file, "asset/post_img/"+strconv.Itoa(int(id))+".png")

	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "create adopt article success",
	})
}

func allAdoptArticlesHandler(c *gin.Context) {

	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
	}
	fmt.Println("page=", page)
	var pageShowNum int = 12

	rows, err := db.Query("SELECT id, author_id, title, city, area, cat_name, cat_age, cat_personality, cat_story FROM cat_adopt_posts order by id desc offset $1 limit $2", (page-1)*pageShowNum, pageShowNum)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
	}
	var adoptPosts []model.AdoptPostModel
	for rows.Next() {
		var adoptPost model.AdoptPostModel
		rows.Scan(&adoptPost.Id, &adoptPost.Author_id, &adoptPost.Title, &adoptPost.City, &adoptPost.Area, &adoptPost.Cat_name, &adoptPost.Cat_age, &adoptPost.Cat_personality, &adoptPost.Cat_story)
		adoptPosts = append(adoptPosts, adoptPost)
	}

	var totalCount int
	row := db.QueryRow("select count(*) from cat_adopt_posts")
	err = row.Scan(&totalCount)
	totalPage := float64(totalCount) / float64(pageShowNum)

	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"errorCode":   0,
		"message":     "ok",
		"adoptPosts":  adoptPosts,
		"currentPage": page,
		"totalPage":   int(math.Ceil(totalPage)),
	})

}
func postAdoptArticleHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "postAdoptArticle.html", nil)
}
func checkSession(c *gin.Context) {

	session := sessions.Default(c)
	sessionID := session.Get("loginuser")

	if sessionID == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "unauthorized",
		})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{
		"message": sessionID,
	})
}
func GetIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "cat_new.html", nil)
}

func loginHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func HelloWorld(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "HelloWorld",
	})
}

func handleGoogleLogin(c *gin.Context) {
	//randomState := "randomState"
	url := googleOauthConfig.AuthCodeURL(randomState)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func callbackHandler(c *gin.Context) {
	if c.Request.FormValue("state") != randomState {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	token, err := googleOauthConfig.Exchange(context.Background(), c.Request.FormValue("code"))
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	//resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}
	fmt.Println("content=", string(content))
	type userInfoStruct struct {
		Email string
		Name  string
	}
	var userInfo userInfoStruct
	json.Unmarshal(content, &userInfo)
	session := sessions.Default(c)
	session.Set("loginuser", userInfo.Email)

	// insert google login user
	var userId int
	db.Exec(`INSERT INTO users(username, password) VALUES ($1, $2)`, userInfo.Email, "")
	db.QueryRow("select id from users where username=$1", userInfo.Email).Scan(&userId)
	session.Set("userId", userId)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func loggedInHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("loginuser")
	session.Clear()
	session.Save()
	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "logged out successfully",
	})
}
func checkLoggedInHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("loginuser")

	if username == nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   "not logged in",
		})
		return
	}
	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "logged in",
		"username":  username,
	})
}
func verifyUserHandler(c *gin.Context) {
	var verifyUserReq model.VerifyUserRequest
	err := c.ShouldBindJSON(&verifyUserReq)
	if err != nil {
		c.JSON(400, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	fmt.Println("verifyUserReq.Name =", verifyUserReq.Name)
	fmt.Println("verifyUserReq.Password =", verifyUserReq.Password)
	rows, err := db.Query(`select id from users where username=$1 and password=$2 `, verifyUserReq.Name, verifyUserReq.Password)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}
	if !rows.Next() {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   "verify fails",
		})
		return
	}
	var userId int
	rows.Scan(&userId)
	fmt.Println("userId = ", userId)
	session := sessions.Default(c)
	session.Set("loginuser", verifyUserReq.Name)
	session.Set("userId", userId)
	session.Save()
	// c.SetCookie("username", verifyUserReq.Name, 3600, "/", "localhost", false, true)
	// c.SetCookie("password", verifyUserReq.Password, 3600, "/", "localhost", false, true)
	// c.SetCookie("catLoginOK", "catLoginOK", 3600, "/", "localhost", false, true)
	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "verify ok",
	})

}
func createUserHandler(c *gin.Context) {
	var createUserReq model.CreateUserRequest
	err := c.ShouldBindJSON(&createUserReq)
	if err != nil {
		c.JSON(400, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	_, err = db.Exec(`INSERT INTO users(username, password) VALUES ($1, $2)`, createUserReq.Username, createUserReq.Password)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "createUser success",
	})
}

func modifyUserHandler(c *gin.Context) {
	var modifyUserReq model.ModifyUserRequest
	err := c.ShouldBindJSON(&modifyUserReq)
	if err != nil {
		c.JSON(400, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	_, err = db.Exec(`update users set username=$1, password=$2 where id=$3 `, modifyUserReq.Username, modifyUserReq.Password, modifyUserReq.Id)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "update User success",
	})
}

func queryUserHandler(c *gin.Context) {
	var myUser model.User
	userSql := "SELECT id, username, password FROM users WHERE id = $1"

	err := db.QueryRow(userSql, 1).Scan(&myUser.Id, &myUser.Name, &myUser.Password)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
	}

	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "query User success",
		"user":      myUser,
	})

}
