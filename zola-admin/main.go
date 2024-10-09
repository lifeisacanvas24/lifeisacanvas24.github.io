package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "github.com/dgrijalva/jwt-go"
    "golang.org/x/crypto/bcrypt"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
    // Initialize database
    var err error
    db, err = sql.Open("sqlite3", "./database.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    app := fiber.New()

    // Serve static files
    app.Static("/css", "./static/css")

    // Routes
    app.Post("/register", registerUser)
    app.Post("/login", loginUser)

    // Protect these routes with authentication middleware
    app.Use(AuthMiddleware)

    app.Get("/admin", adminDashboard)           // Admin dashboard
    app.Get("/products", getProducts)           // Get products
    app.Get("/blog/posts", getBlogPosts)        // Get blog posts

    // Start server
    log.Fatal(app.Listen(":3000"))
}

// User Registration Handler
func registerUser(c *fiber.Ctx) error {
    type User struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    user := new(User)
    if err := c.BodyParser(user); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid input",
        })
    }

    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 14)

    // Insert into database
    _, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user.Username, hashedPassword)
    if err != nil {
        return c.Status(fiber.StatusConflict).JSON(fiber.Map{
            "error": "Username already exists",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User registered successfully",
    })
}

// User Login Handler
func loginUser(c *fiber.Ctx) error {
    type Login struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    login := new(Login)
    if err := c.BodyParser(login); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid input",
        })
    }

    var hashedPassword string
    err := db.QueryRow("SELECT password FROM users WHERE username = ?", login.Username).Scan(&hashedPassword)
    if err != nil || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(login.Password)) != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid username or password",
        })
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "id": login.Username, // You can also store user ID here
    })

    tokenString, err := token.SignedString([]byte("your_secret_key")) // Replace with your secret key
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Could not generate token",
        })
    }

    return c.JSON(fiber.Map{
        "token": tokenString,
    })
}

// Authentication Middleware
func AuthMiddleware(c *fiber.Ctx) error {
    token := c.Get("Authorization")

    if token == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Unauthorized",
        })
    }

    claims := jwt.MapClaims{}
    _, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
        return []byte("your_secret_key"), nil
    })

    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Unauthorized",
        })
    }

    c.Locals("userId", claims["id"])
    return c.Next()
}

// Admin dashboard handler
func adminDashboard(c *fiber.Ctx) error {
    return c.Render("admin.html", nil)
}

// Get products handler
func getProducts(c *fiber.Ctx) error {
    rows, err := db.Query("SELECT * FROM products")
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Could not fetch products",
        })
    }
    defer rows.Close()

    var products []map[string]interface{}
    for rows.Next() {
        var id int
        var name string
        var price float64
        var description string
        var stock int
        if err := rows.Scan(&id, &name, &price, &description, &stock); err != nil {
            return err
        }
        products = append(products, map[string]interface{}{
            "id":          id,
            "name":        name,
            "price":       price,
            "description": description,
            "stock":       stock,
        })
    }

    return c.JSON(products)
}

// Get blog posts handler
func getBlogPosts(c *fiber.Ctx) error {
    rows, err := db.Query("SELECT * FROM blog_posts")
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Could not fetch blog posts",
        })
    }
    defer rows.Close()

    var posts []map[string]interface{}
    for rows.Next() {
        var id int
        var title string
        var content string
        var category string
        var rating int
        if err := rows.Scan(&id, &title, &content, &category, &rating); err != nil {
            return err
        }
        posts = append(posts, map[string]interface{}{
            "id":       id,
            "title":    title,
            "content":  content,
            "category": category,
            "rating":   rating,
        })
    }

    return c.JSON(posts)
}
