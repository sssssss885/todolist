package service

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	database "todolist.go/db"
)

// TaskList renders list of tasks in DB
func TaskList(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // Get query parameter
    kw := ctx.Query("kw")
	is_done := ctx.Query("is_done")
	is_not_done := ctx.Query("is_not_done")
 
    // Get tasks in DB
    var tasks []database.Task
	query := "SELECT id, title, created_at, is_done FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"
    switch {
    case is_done=="t" && is_not_done=="f":
        err = db.Select(&tasks, query + " AND title LIKE ?", userID, "%" + kw + "%")
	case is_done=="t":
		err = db.Select(&tasks, query + " AND is_done = 1 AND title LIKE ?", userID,  "%" + kw + "%")
	case is_not_done=="f":
		err = db.Select(&tasks, query + " AND is_done = 0 AND title LIKE ?", userID,  "%" + kw + "%")
    default:
        err = db.Select(&tasks, "SELECT * FROM tasks WHERE 1=0")
    }
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // Render tasks
    ctx.HTML(http.StatusOK, "task_list.html", gin.H{"Title": "Task list", "Tasks": tasks, "Kw": kw, "Is_Done": is_done, "Is_Not_Done": is_not_done})
}

// ShowTask renders a task with given ID
func ShowTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// parse ID given as a parameter
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// Get a task with given ID
	var task database.Task
	query := "SELECT id, title, created_at, is_done FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"
	err = db.Get(&task, query + " AND id=?", userID, id) // Use DB#Get for one entry
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// Render task
	//ctx.String(http.StatusOK, task.Title)  // Modify it!!
	ctx.HTML(http.StatusOK, "task.html", task)
}

func NewTaskForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "form_new_task.html", gin.H{"Title": "Task registration"})
}

func RegisterTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
    // Get task title
    title, exist := ctx.GetPostForm("title")
    if !exist {
        Error(http.StatusBadRequest, "No title is given")(ctx)
        return
    }

    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    tx := db.MustBegin()
    result, err := tx.Exec("INSERT INTO tasks (title) VALUES (?)", title)
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    taskID, err := result.LastInsertId()
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    _, err = tx.Exec("INSERT INTO ownership (user_id, task_id) VALUES (?, ?)", userID, taskID)
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    tx.Commit()
	
    ctx.Redirect(http.StatusFound, fmt.Sprintf("/task/%d", taskID))
}

func EditTaskForm(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
    //task ID の取得
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    // Get target task
    var task database.Task
	query := "SELECT id, title, created_at, is_done FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"
    err = db.Get(&task, query + " AND id=?", userID, id)
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

    // Render edit form
    ctx.HTML(http.StatusOK, "form_edit_task.html",
        gin.H{"Title": fmt.Sprintf("Edit task %d", task.ID), "Task": task})
}

func UpdateTask(ctx *gin.Context) {
	//Get task title
	title, exist := ctx.GetPostForm("title")
	if !exist {
		Error(http.StatusBadRequest, "No title is given")(ctx)
		return
	}

	//Get task is_done
	is_done_before, exist1 := ctx.GetPostForm("is_done")
	is_done, _ := strconv.ParseBool(is_done_before)
	if !exist1 {
		Error(http.StatusBadRequest, "No is_done is given")(ctx)
		return
	}

	//Get ID
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}
	
	//Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	
	//Update target task
	//var task database.Task
	_, err = db.Exec("UPDATE tasks SET title=?, is_done=? WHERE id=?", title, is_done, id)
	if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
	//Render status
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/task/%d", id))
}

func DeleteTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
    //task ID の取得
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

	//deleteを要求されたtaskのuserIDとsessionsのuserIDが一致しない場合にエラー
	var task database.Task
	tx := db.MustBegin()
	query := "SELECT id, title, created_at, is_done FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"
    err = db.Get(&task, query + " AND id=?", userID, id)
    if err != nil {
		tx.Rollback()
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

    // Delete the task from DB
    _, err = db.Exec("DELETE FROM tasks WHERE id=?", id)
    if err != nil {
		tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

	// Delete the ownership from DB
	_, err = db.Exec("DELETE FROM ownership WHERE task_id=?", id)
    if err != nil {
		tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
	tx.Commit()

    // Redirect to /list
    ctx.Redirect(http.StatusFound, "/list?kw=&is_done=t&is_not_done=f")
}