package main

import (
	"database/sql"  // Database SQL package to perform queries
	"log"           // Display messages to console
	"net/http"      // Manage URL
	"text/template" // Manage HTML files

	_ "github.com/go-sql-driver/mysql" // MySQL Database driver
	"github.com/gorilla/mux"
	_ "github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	_ "github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
)

// cookie handling

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func getUserName(request *http.Request) (userName string) {
	if cookie, err := request.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["name"]
		}
	}
	return userName
}

func setSession(userName string, response http.ResponseWriter) {
	value := map[string]string{
		"name": userName,
	}
	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
	}
}

func clearSession(response http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(response, cookie)
}

// login handler

func loginHandler(res http.ResponseWriter, req *http.Request) {
	username := req.FormValue("name")
	password := req.FormValue("password")
	redirectTarget := "/"
	if username != "" && password != "" {

		db := dbConn()

		var databaseUsername string
		var databasePassword string

		err := db.QueryRow("SELECT username, password FROM users WHERE username=?", username).Scan(&databaseUsername, &databasePassword)

		// Call the struct to be rendered on template and Create a slice to store all data from struct
		var msg = "Usuario y/o password incorrectos"
		if err != nil {
			defer db.Close()
			tmpl.ExecuteTemplate(res, "IndexLogin", msg)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(databasePassword), []byte(password))
		if err != nil {
			defer db.Close()
			tmpl.ExecuteTemplate(res, "IndexLogin", msg)
			return
		}

		setSession(username, res)
		redirectTarget = "/index"
		// Close database connection
		defer db.Close()
	}
	http.Redirect(res, req, redirectTarget, 302)
}

// logout handler

func logoutHandler(response http.ResponseWriter, request *http.Request) {
	clearSession(response)
	http.Redirect(response, request, "/", 302)
}

func indexPageHandler(response http.ResponseWriter, request *http.Request) {
	if getUserName(request) != "" {
		http.Redirect(response, request, "/index", 302)
	}
	var msg = ""
	tmpl.ExecuteTemplate(response, "IndexLogin", msg)
}

// Struct used to send data to template
// this struct is the same as the database
type Names struct {
	Id    int
	Name  string
	Email string
}

// Function dbConn opens connection with MySQL driver
// send the parameter `db *sql.DB` to be used by another functions
func dbConn() (db *sql.DB) {

	dbDriver := "mysql" // Database driver
	dbUser := "root"    // Mysql username
	dbPass := "root"    // Mysql password
	dbName := "base"    // Mysql schema

	// Realize the connection with mysql driver
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)

	// If error stop the application
	if err != nil {
		panic(err.Error())
	}

	// Return db object to be used by other functions
	return db
}

// Read all templates on folder `tmpl/*`
var tmpl = template.Must(template.ParseGlob("tmpl/*"))

func checkSession(w http.ResponseWriter, r *http.Request) {
	userName := getUserName(r)
	if userName == "" {
		http.Redirect(w, r, "/", 302)
	}
}

// Function Index shows all values on home
func Index(w http.ResponseWriter, r *http.Request) {

	checkSession(w, r)

	// Open database connection
	db := dbConn()

	// Prepare a SQL query to select all data from database and threat errors
	selDB, err := db.Query("SELECT * FROM names ORDER BY id DESC")
	if err != nil {
		panic(err.Error())
	}

	// Call the struct to be rendered on template
	n := Names{}

	// Create a slice to store all data from struct
	res := []Names{}

	// Read all rows from database
	for selDB.Next() {
		// Must create this variables to store temporary query
		var id int
		var name, email string

		// Scan each row storing values from the variables above and check for errors
		err = selDB.Scan(&id, &name, &email)
		if err != nil {
			panic(err.Error())
		}

		// Get the Scan into the Struct
		n.Id = id
		n.Name = name
		n.Email = email

		// Join each row on struct inside the Slice
		res = append(res, n)
	}

	// Execute template `Index` from `tmpl/*` folder and send the struct
	// (View the file: `tmpl/Index`
	tmpl.ExecuteTemplate(w, "Index", res)

	// Close database connection
	defer db.Close()

}

// Function Show displays a single value
func Show(w http.ResponseWriter, r *http.Request) {

	checkSession(w, r)
	// Open database connection
	db := dbConn()

	// Get the URL `?id=X` parameter
	nId := r.URL.Query().Get("id")

	// Perform a SELECT query getting the register Id(See above) and check for errors
	selDB, err := db.Query("SELECT * FROM names WHERE id=?", nId)
	if err != nil {
		panic(err.Error())
	}

	// Call the struct to be rendered on template
	n := Names{}

	// Read all rows from database
	// This time we are going to get only one value, doesn't need the slice
	for selDB.Next() {
		// Store query values on this temporary variables
		var id int
		var name, email string

		// Scan each row to match the ID and check for errors
		err = selDB.Scan(&id, &name, &email)
		if err != nil {
			panic(err.Error())
		}

		// Get the Scan into the Struct
		n.Id = id
		n.Name = name
		n.Email = email

	}

	// Execute template `Show` from `tmpl/*` folder and send the struct
	// (View the file: `tmpl/Show`)
	tmpl.ExecuteTemplate(w, "Show", n)

	// Close database connection
	defer db.Close()

}

// Function New just parse a form to send data to Insert function
// (View the file: `tmpl/New`)
func New(w http.ResponseWriter, r *http.Request) {
	checkSession(w, r)
	tmpl.ExecuteTemplate(w, "New", nil)
}

// Function Edit works like Show
// Only select the values to send to the Edit page Form
// (View the file: `tmpl/Edit`)
func Edit(w http.ResponseWriter, r *http.Request) {
	checkSession(w, r)
	db := dbConn()

	// Get the URL `?id=X` parameter
	nId := r.URL.Query().Get("id")

	selDB, err := db.Query("SELECT * FROM names WHERE id=?", nId)
	if err != nil {
		panic(err.Error())
	}

	n := Names{}

	for selDB.Next() {
		var id int
		var name, email string

		err = selDB.Scan(&id, &name, &email)
		if err != nil {
			panic(err.Error())
		}

		n.Id = id
		n.Name = name
		n.Email = email

	}

	tmpl.ExecuteTemplate(w, "Edit", n)

	defer db.Close()
}

// Function Insert puts data into the database
func Insert(w http.ResponseWriter, r *http.Request) {
	checkSession(w, r)
	// Open database connection
	db := dbConn()

	// Check the request form METHOD
	if r.Method == "POST" {

		// Get the values from Form
		name := r.FormValue("name")
		email := r.FormValue("email")

		// Prepare a SQL INSERT and check for errors
		insForm, err := db.Prepare("INSERT INTO names(name, email) VALUES(?,?)")
		if err != nil {
			panic(err.Error())
		}

		// Execute the prepared SQL, getting the form fields
		insForm.Exec(name, email)

		// Show on console the action
		log.Println("INSERT: Name: " + name + " | E-mail: " + email)
	}

	// Close database connection
	defer db.Close()

	// Redirect to HOME
	http.Redirect(w, r, "/index", 301)
}

func IndexSignUp(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		tmpl.ExecuteTemplate(res, "IndexSignUp", req)
		return
	}

	db := dbConn()

	username := req.FormValue("name")
	password := req.FormValue("password")

	var user string

	err := db.QueryRow("SELECT username FROM users WHERE username=?", username).Scan(&user)

	switch {
	case err == sql.ErrNoRows:
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(res, "Server error, unable to create your account.", 500)
			return
		}

		_, err = db.Exec("INSERT INTO users(username, password) VALUES(?, ?)", username, hashedPassword)
		if err != nil {
			http.Error(res, "Server error, unable to create your account.", 500)
			return
		}

		res.Write([]byte("User created!"))
		return
	case err != nil:
		http.Error(res, "Server error, unable to create your account.", 500)
		return
	default:
		http.Redirect(res, req, "/", 301)
	}
}

// Function Update, update values from database,
// It's the same as Insert and New
func Update(w http.ResponseWriter, r *http.Request) {

	checkSession(w, r)
	db := dbConn()

	if r.Method == "POST" {

		// Get the values from form
		name := r.FormValue("name")
		email := r.FormValue("email")
		id := r.FormValue("uid") // This line is a hidden field on form (View the file: `tmpl/Edit`)

		// Prepare the SQL Update
		insForm, err := db.Prepare("UPDATE names SET name=?, email=? WHERE id=?")
		if err != nil {
			panic(err.Error())
		}

		// Update row based on hidden form field ID
		insForm.Exec(name, email, id)

		// Show on console the action
		log.Println("UPDATE: Name: " + name + " | E-mail: " + email)
	}

	defer db.Close()

	// Redirect to Home
	http.Redirect(w, r, "/index", 301)
}

// Function Delete destroys a row based on ID
func Delete(w http.ResponseWriter, r *http.Request) {

	checkSession(w, r)
	db := dbConn()

	// Get the URL `?id=X` parameter
	nId := r.URL.Query().Get("id")

	// Prepare the SQL Delete
	delForm, err := db.Prepare("DELETE FROM names WHERE id=?")
	if err != nil {
		panic(err.Error())
	}

	// Execute the Delete SQL
	delForm.Exec(nId)

	// Show on console the action
	log.Println("DELETE")

	defer db.Close()

	// Redirect a HOME
	http.Redirect(w, r, "/index", 301)
}

var router = mux.NewRouter()

func main() {

	// Show on console the application stated
	log.Println("Server started on: http://localhost:8080")

	// URL management
	// Manage templates
	router.HandleFunc("/", indexPageHandler) // LOGIN :: Show all registers
	router.HandleFunc("/index", Index)       // INDEX :: Show all registers
	router.HandleFunc("/show", Show)         // SHOW  :: Show only one register
	router.HandleFunc("/new", New)           // NEW   :: Form to create new register
	router.HandleFunc("/edit", Edit)         // EDIT  :: Form to edit register
	router.HandleFunc("/signup", IndexSignUp)

	// Manage actions
	router.HandleFunc("/insert", Insert) // INSERT :: New register
	router.HandleFunc("/update", Update) // UPDATE :: Update register
	router.HandleFunc("/delete", Delete) // DELETE :: Destroy register

	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/logout", logoutHandler) //LOGOUT

	http.Handle("/", router)
	// Start the server on port 9000
	http.ListenAndServe(":8080", nil)

}
