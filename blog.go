package main

import (
    "fmt"
    "log"
    "net/http"
    "time"
    "context"
    "html/template"
    "strconv"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func getClient() *mongo.Client {
    clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
    client, err := mongo.Connect(context.TODO(), clientOptions)
    if err != nil {
        log.Fatal(err)
    }
    return client
}

type Post struct {
    ID int
    Title string
    Body string
    Date time.Time
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, *mongo.Client), c *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        fn(w, r, c)
    }
}

func (post *Post) getPostId(client *mongo.Client) {
    collection := client.Database("blog").Collection("posts")
    c, err := collection.CountDocuments(context.TODO(), bson.M{})
    if err != nil {
        log.Fatal(err)
    }
    id := int(c)
    post.ID = id
    
}

func handleHome(w http.ResponseWriter, r *http.Request, c *mongo.Client) {
    posts := getAllPosts(c, bson.M{})
    tmp, err := template.ParseFiles("./tmpls/base.html", "./tmpls/index.html")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
    tmp.Execute(w, posts)
}

func newPostHandler(w http.ResponseWriter, r *http.Request) {
    newPost := Post{0, "", "", time.Now()}
    tmp, err := template.ParseFiles("./tmpls/base.html", "./tmpls/new_post.html")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
    tmp.Execute(w, newPost)
}

func saveHandler(w http.ResponseWriter, r *http.Request, client *mongo.Client) {
    collection := client.Database("blog").Collection("posts")
    title := r.FormValue("title")
    body := r.FormValue("body")
    date := time.Now()
    post := Post{0, title, body, date}
    post.getPostId(client)
    _, err := collection.InsertOne(context.TODO(), post)
    if err != nil {
        log.Fatal(err)
    }
    http.Redirect(w, r, "/", http.StatusFound)
}

func viewPostHandler(w http.ResponseWriter, r *http.Request, client *mongo.Client) {
    var post *Post
    collection := client.Database("blog").Collection("posts")
    post_id := string(r.URL.String()[6:])
    fmt.Println(post_id)
    id, err := strconv.Atoi(post_id)
    if err != nil {
        log.Fatal("could not convert id to int")
    }
    found := collection.FindOne(context.TODO(), bson.D{{"id", id}}) 
    err = found.Decode(&post)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
    fmt.Println(post.Title)
    template, err := template.ParseFiles("tmpls/base.html", "tmpls/post.html")
    template.Execute(w, post)

}

func getAllPosts(client *mongo.Client, filter bson.M) []*Post {
    findOptions := options.Find()
    findOptions.SetSort(bson.D{{"_id", -1}})
    var posts []*Post
    collection := client.Database("blog").Collection("posts")
    cur, err := collection.Find(context.TODO(), filter, findOptions)
    if err != nil {
        log.Fatal("could not find posts", err)
    }
    for cur.Next(context.TODO()) {
        var post Post
        err = cur.Decode(&post)
        if err != nil {
            log.Fatal("Error decoding post document", err)
        }
        posts = append(posts, &post)
    }
    return posts
}

func main() {
    client := getClient()
    fmt.Println("Server running on port 8080")
    fmt.Println(time.Now())
    //index := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    http.HandleFunc("/", makeHandler(handleHome, client))
    http.HandleFunc("/new_post/", newPostHandler)
    http.HandleFunc("/save/", makeHandler(saveHandler, client))
    http.HandleFunc("/post/", makeHandler(viewPostHandler, client))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
