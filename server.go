package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"goWeb/model"
	"goWeb/ws"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	// Initialize connection constants.
	HOST     = os.Getenv("HOST")
	DATABASE = os.Getenv("DATABASE")
	USER     = os.Getenv("USER")
	PASSWORD = os.Getenv("PASSWORD")
)

var (
	db                *sql.DB
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("RedirectURL"),
		ClientID:     os.Getenv("ClientID"),
		ClientSecret: os.Getenv("ClientSecret"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
	randomState = "random"
)

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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
	go ws.Manager.Start()

	r := gin.Default()
	r.LoadHTMLGlob("view/*")
	r.Static("asset", "./asset")

	store := sessions.NewCookieStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	r.GET("/ping", HelloWorld)
	r.GET("/", GetIndex)
	r.GET("/checkSession", checkSession)
	r.GET("/loginGoogle", handleGoogleLogin)
	r.GET("/callback", callbackHandler)
	r.GET("/postAdoptArticle", postAdoptArticleHandler)

	r.GET("/catHTMLVue", catHTMLVueJSHandler)
	r.GET("/catInfo", catInfoHandler)
	r.GET("/getCatImg", getCatImgHandler)
	r.GET("/managePosts", managePostsHandler)

	r.GET("/about", aboutHandler)

	r.GET("/allAdoptArticles", allAdoptArticlesHandler)

	r.POST("/createUser", createUserHandler)
	r.POST("/modifyUser", modifyUserHandler)
	r.POST("/verifyUser", verifyUserHandler)
	r.POST("/logout", logOutHandler)

	r.POST("/createAdoptArticle", createAdoptArticleHandler)

	r.GET("/getUserPosts", getUserPostsHandler)

	r.GET("/User", queryUserHandler)
	r.GET("/login", loginHandler)
	r.GET("/checkLoggedIn", checkLoggedInHandler)

	r.GET("/webSocket", WsPing)
	r.GET("/chat", chatHTML)
	r.GET("/ws", ws.WsHandler)

	r.DELETE("/deleteAdoptPost", deleteAdoptPostHandler)

	r.PUT("/updateAdoptPost", updateAdoptPostHandler)
	r.Run() // listen and serve (for windows "http://localhost:8080")
}

func WsPing(ctx *gin.Context) {

	ws, err := upGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	for {
		// ??????ws Socket???????????????
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		// ??????Websocket
		err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintln("got it : "+string(message))))
		if err != nil {
			break
		}
	}
}

func chatHTML(c *gin.Context) {
	c.HTML(http.StatusOK, "chat_contact.html", nil)
}
func catHTMLVueJSHandler(c *gin.Context) {
	postIdStr := c.Query("postId")
	if postIdStr == "" {
		postIdStr = "1"
	}
	postId, err := strconv.Atoi(postIdStr)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	session := sessions.Default(c)
	session.Set("postId", postId)
	session.Save()

	c.HTML(http.StatusOK, "cat_info_vueJS.html", nil)
}

func aboutHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "about.html", nil)
}

func getCatImgHandler(c *gin.Context) {
	postIdStr := c.Query("postId")
	if postIdStr == "" {
		postIdStr = "1"
	}
	postId, err := strconv.Atoi(postIdStr)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}
	var imgBytes []byte
	err = db.QueryRow("select img_src from cat_adopt_posts where id=$1", postId).Scan(&imgBytes)

	//fmt.Println("imgBytes=", imgBytes)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", len(imgBytes)))
	c.Writer.Write(imgBytes)

}
func catInfoHandler(c *gin.Context) {

	session := sessions.Default(c)
	postIdSession := session.Get("postId")
	postId := 1
	if postIdSession != nil {
		postId = postIdSession.(int)
	}

	var catPost model.AdoptPostModel
	row := db.QueryRow("select id,author_id, title, city, area, cat_name, cat_age,cat_personality,cat_story, contact_info  from cat_adopt_posts where id =$1", postId)
	err := row.Scan(&catPost.Id, &catPost.Author_id, &catPost.Title, &catPost.City, &catPost.Area, &catPost.Cat_name, &catPost.Cat_age, &catPost.Cat_personality, &catPost.Cat_story, &catPost.Contact_info)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
	}
	var authorName string
	db.QueryRow("select name from users where id=$1", catPost.Author_id).Scan(&authorName)
	c.JSON(200, gin.H{
		"errorCode":  0,
		"catPost":    catPost,
		"authorName": authorName,
	})
}

func getUserPostsHandler(c *gin.Context) {
	session := sessions.Default(c)
	//sessionID := session.Get("loginuser")
	userID := session.Get("userId")
	fmt.Println("userid=", userID)
	if userID == nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   "????????????",
		})
		return
	}

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

	rows, err := db.Query("SELECT id, author_id, title, city, area, cat_name, cat_age, cat_personality, cat_story  , contact_info FROM cat_adopt_posts where author_id=$3 order by id desc offset $1 limit $2", (page-1)*pageShowNum, pageShowNum, userID.(int))
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
	}
	var adoptPosts []model.AdoptPostModel
	for rows.Next() {
		var adoptPost model.AdoptPostModel
		rows.Scan(&adoptPost.Id, &adoptPost.Author_id, &adoptPost.Title, &adoptPost.City, &adoptPost.Area, &adoptPost.Cat_name, &adoptPost.Cat_age, &adoptPost.Cat_personality, &adoptPost.Cat_story, &adoptPost.Contact_info)
		adoptPosts = append(adoptPosts, adoptPost)
	}

	var totalCount int
	row := db.QueryRow("select count(*) from cat_adopt_posts where author_id=$1", userID.(int))
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

func updateAdoptPostHandler(c *gin.Context) {

	file, _ := c.FormFile("catPicture")
	sql := ""
	var byteContainer []byte
	if file != nil {
		fileContent, _ := file.Open()
		byteContainer, _ = ioutil.ReadAll(fileContent)
		sql = "update cat_adopt_posts set title=$1, city=$2, area=$3, cat_name=$4, cat_age=$5, cat_personality=$6, cat_story=$7, contact_info=$8, img_src=$9 where id=$10"
		_, err := db.Exec(sql, c.PostForm("title"), c.PostForm("city"), c.PostForm("area"), c.PostForm("catName"), c.PostForm("catAge"), c.PostForm("catPersonality"), c.PostForm("catStory"), c.PostForm("contactInfo"), byteContainer, c.PostForm("postId"))
		if err != nil {
			c.JSON(200, gin.H{
				"errorCode": -1,
				"message":   err.Error(),
			})
			return
		}
	} else {
		fmt.Println("i am herrrr")
		sql = "update cat_adopt_posts set title=$1, city=$2, area=$3, cat_name=$4, cat_age=$5, cat_personality=$6, cat_story=$7, contact_info=$8 where id=$9"
		_, err := db.Exec(sql, c.PostForm("title"), c.PostForm("city"), c.PostForm("area"), c.PostForm("catName"), c.PostForm("catAge"), c.PostForm("catPersonality"), c.PostForm("catStory"), c.PostForm("contactInfo"), c.PostForm("postId"))
		if err != nil {
			fmt.Println("i an hhhhhh")
			c.JSON(200, gin.H{
				"errorCode": -1,
				"message":   err.Error(),
			})
			return
		}
	}

	c.JSON(200, gin.H{
		"errorCode": 0,
		"message":   "update adopt article success",
	})
}
func deleteAdoptPostHandler(c *gin.Context) {
	var postId struct {
		PostId int
	}
	c.ShouldBindJSON(&postId)
	fmt.Println("postId=", postId.PostId)

	_, err := db.Exec("delete from cat_adopt_posts where id =$1", postId.PostId)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"errorCode":     0,
		"message":       "delete adopt article success",
		"deletedPostId": postId.PostId,
	})
}
func createAdoptArticleHandler(c *gin.Context) {

	session := sessions.Default(c)
	//sessionID := session.Get("loginuser")
	userID := session.Get("userId")
	fmt.Println("userid=", userID)
	if userID == nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   "????????????",
		})
		return
	}

	var id int
	file, err := c.FormFile("catPicture")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}
	fileContent, _ := file.Open()
	byteContainer, _ := ioutil.ReadAll(fileContent)
	// c.SaveUploadedFile(file, "asset/post_img/"+strconv.Itoa(int(id))+".png")

	err = db.QueryRow(`INSERT INTO cat_adopt_posts(city, area,cat_name,cat_age,cat_personality,cat_story,title,author_id,contact_info,img_src) VALUES ($1, $2, $3,$4,$5,$6,$7,$8,$9,$10) returning id`, c.PostForm("city"), c.PostForm("area"), c.PostForm("catName"), c.PostForm("catAge"), c.PostForm("catPersonality"), c.PostForm("catStory"), c.PostForm("title"), userID.(int), c.PostForm("contactInfo"), byteContainer).Scan(&id)
	if err != nil {
		c.JSON(200, gin.H{
			"errorCode": -1,
			"message":   err.Error(),
		})
		return
	}

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

func managePostsHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "managePosts.html", nil)
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
		"message": "HelloWorld test airdd",
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
	fmt.Println("########## content =", string(content))
	json.Unmarshal(content, &userInfo)
	session := sessions.Default(c)
	session.Set("loginuser", userInfo.Email)

	// insert google login user
	var userId int
	db.Exec(`INSERT INTO users(username, password,name) VALUES ($1, $2,$3)`, userInfo.Email, "123123", userInfo.Name)
	db.QueryRow("select id from users where username=$1", userInfo.Email).Scan(&userId)
	session.Set("userId", userId)
	session.Set("username", userInfo.Email)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func logOutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("loginuser")
	session.Delete("userId")
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
	name := ""
	db.QueryRow("SELECT NAME FROM users where username=$1", username).Scan(&name)

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
		"name":      name,
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
