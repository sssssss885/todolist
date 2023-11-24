package service
 
import (
    "fmt"
    "strconv"
    "regexp"
	"crypto/sha256"
    "encoding/hex"
    "net/http"
    "unicode/utf8"
 
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/sessions"
	database "todolist.go/db"
)
 
func NewUserForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "new_user_form.html", gin.H{"Title": "Register user"})
}

func hash(pw string) []byte {
    const salt = "todolist.go#"
    h := sha256.New()
    h.Write([]byte(salt))
    h.Write([]byte(pw))
    return h.Sum(nil)
}

func RegisterUser(ctx *gin.Context) {
    // フォームデータの受け取り
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
    repassword := ctx.PostForm("repassword")
    switch {
    case username == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Usernane is not provided", "Username": username})
    case password == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Password is not provided", "Password": password})
    case repassword == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Password is not provided", "Password": repassword})
    }
    
    // DB 接続
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

	// 重複チェック
    var duplicate int
    err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name=?", username)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    if duplicate > 0 {
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Username is already taken", "Username": username, "Password": password})
        return
    }

    //passwordとrepasswordが一致しない場合
    if password!=repassword {
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Enter the same password", "Username": username, "Password": password})
        return
    }

    //簡単なパスワードを拒否(文字が少ない)
    if 8 > utf8.RuneCountInString(password) {
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "password : Enter at least 8 characters", "Username": username, "Password": password})
        return
    }

    //簡単なパスワードを拒否(数字しか使われていない)
    rex := regexp.MustCompile("[0-9]+")
    password_only_num := rex.FindString(password)
    _, err = strconv.ParseInt(password_only_num, 10, 64)
    if err != nil {
        fmt.Println(err)
    }
    if password == password_only_num {
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "password : Use more than just numbers", "Username": username, "Password": password})
        return
    }
 
    // DB への保存
    result, err := db.Exec("INSERT INTO users(name, password) VALUES (?, ?)", username, hash(password))
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // 保存状態の確認
    id, _ := result.LastInsertId()
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE id = ?", id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    //ctx.JSON(http.StatusOK, user)
    ctx.HTML(http.StatusOK, "index.html", gin.H{"Title": "Register user"})
}

func LoginUserForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "login.html", gin.H{"Title": "Login"})
}

const userkey = "user"
 
func Login(ctx *gin.Context) {
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
 
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE name = ?", username)
    if err != nil {
        ctx.HTML(http.StatusBadRequest, "login.html", gin.H{"Title": "Login", "Username": username, "Error": "No such user"})
        return
    }
 
    // パスワードの照合
    if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(password)) {
        ctx.HTML(http.StatusBadRequest, "login.html", gin.H{"Title": "Login", "Username": username, "Error": "Incorrect password"})
        return
    }
 
    // セッションの保存
    session := sessions.Default(ctx)
    session.Set(userkey, user.ID)
    session.Save()
 
    ctx.Redirect(http.StatusFound, "/list?kw=&is_done=t&is_not_done=f")
}

func LoginCheck(ctx *gin.Context) {
    if sessions.Default(ctx).Get(userkey) == nil {
        ctx.Redirect(http.StatusFound, "/login")
        ctx.Abort()
    } else {
        ctx.Next()
    }
}

func Logout(ctx *gin.Context) {
    session := sessions.Default(ctx)
    session.Clear()
    session.Options(sessions.Options{MaxAge: -1})
    session.Save()
    ctx.Redirect(http.StatusFound, "/")
}

//ユーザー確認のためのフォームを表示
func CheckUserForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "form_check_user.html", gin.H{"Title": "Check user"})
}
 
//ユーザー名変更前に本人か確認
func CheckUser(ctx *gin.Context) {
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
 
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE name = ?", username)
    if err != nil {
        ctx.HTML(http.StatusBadRequest, "form_check_user.html", gin.H{"Title": "Check user", "Error": "No such user"})
        return
    }
 
    // パスワードの照合
    if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(password)) {
        ctx.HTML(http.StatusBadRequest, "form_check_user.html", gin.H{"Title": "Check user", "Username": username, "Error": "Incorrect password"})
        return
    }
    ctx.HTML(http.StatusOK, "form_edit_user.html", gin.H{"Title": "Edit User", "Username_Old": username, "Password_Old": password})
}

//ユーザー名・パスワード変更
func UpdateUser(ctx *gin.Context) {
    username_old := ctx.PostForm("username_old")
    password_old := ctx.PostForm("password_old")
    username_new := ctx.PostForm("username_new")
    password_new := ctx.PostForm("password_new")
    repassword_new := ctx.PostForm("repassword_new")

    switch {
    case username_new == "":
        ctx.HTML(http.StatusBadRequest, "form_edit_user_.html", gin.H{"Title": "Edit User", "Error": "Usernane is not provided", "Username_Old": username_old})
    case password_new == "":
        ctx.HTML(http.StatusBadRequest, "form_edit_user_.html", gin.H{"Title": "Edit User", "Error": "Password is not provided", "Password_Old": password_old})
    case repassword_new == "":
        ctx.HTML(http.StatusBadRequest, "form_edit_user_.html", gin.H{"Title": "Edit User", "Error": "Password is not provided", "Password_Old": repassword_new})
    }
    
    // DB 接続
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

	// 重複チェック(usernameを変更した場合)
    if username_old != username_new {
        var duplicate int
        err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name=?", username_new)
        if err != nil {
            Error(http.StatusInternalServerError, err.Error())(ctx)
            return
        }
        if duplicate > 0 {
            ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": "Edit User", "Error": "Username is already taken", "Username_Old": username_old, "Password_Old": password_old})
            return
        }
    }

    //password_newとrepassword_newが一致しない場合
    if password_new!=repassword_new {
        ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": "Edit User", "Error": "Enter the same password", "Username_Old": username_old, "Password_Old": password_old, "Username_New": username_new})
        return
    }

    //簡単なパスワードを拒否(文字が少ない)
    if 8 > utf8.RuneCountInString(password_new) {
        ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": "Edit User", "Error": "password : Enter at least 8 characters", "Username_Old": username_old, "Password_Old": password_old, "Username_New": username_new})
        return
    }

    //簡単なパスワードを拒否(数字しか使われていない)
    rex := regexp.MustCompile("[0-9]+")
    password_only_num := rex.FindString(password_new)
    _, err = strconv.ParseInt(password_only_num, 10, 64)
    if err != nil {
        fmt.Println(err)
    }
    if password_new == password_only_num {
        ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": "Edit User", "Error": "password : Use more than just numbers", "Username_Old": username_old, "Password_Old": password_old, "Username_New": username_new})
        return
    }



    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE name = ?", username_old)
    //Update User data
    _, err = db.Exec("UPDATE users SET name=?, password=? WHERE id=?", username_new, hash(password_new), user.ID)
	if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    ctx.Redirect(http.StatusFound, "/")
}

//ユーザーの削除
func DeleteUser(ctx *gin.Context) {
    userID := sessions.Default(ctx).Get("user")
    
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    //delete user
    _, err = db.Exec("DELETE FROM users WHERE id=?", userID)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    ctx.Redirect(http.StatusFound, "/")
}