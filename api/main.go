// Package main creates and starts a web server
package main

// @APITitle Brand Foods Product Database

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	auth "github.com/prLorence/fdc-api/auth"
	"github.com/prLorence/fdc-api/ds"
	"github.com/prLorence/fdc-api/ds/cb"
	fdc "github.com/prLorence/fdc-api/model"
)

const (
	maxListSize    = 150
	defaultListMax = 50
	apiVersion     = "1.0.0 Beta"
	JSONSPEC       = "./dist/apiDoc.json"
	YAMLSPEC       = "./dist/apiDoc.yaml"
)

var (
	s   = flag.String("s", "dist", "Path for static files")
	i   = flag.String("i", "", "Initialize the authentication store")
	c   = flag.String("c", "config.yml", "YAML Config file")
	l   = flag.String("l", "/tmp/bfpd.out", "send log output to this file -- defaults to /tmp/bfpd.out")
	p   = flag.String("p", "8000", "TCP port to used")
	r   = flag.String("r", "v1", "root path to deploy -- defaults to 'v1'")
	cs  fdc.Config
	err error
	dc  ds.DataSource
)

// process cli flags; build the config and init an Mongo client and a logger
func init() {
	var lfile *os.File
	lfile, err = os.OpenFile(*l, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", *l, ":", err)
	}
	m := io.MultiWriter(lfile, os.Stdout)
	log.SetOutput(m)
}

func main() {
	var cb cb.Cb
	flag.Parse()
	// get configuration
	cs.GetConfig(c)
	// Create a datastore and connect to it
	dc = &cb
	err = dc.ConnectDs(cs)
	if err != nil {
		log.Fatalf("Cannot get datastore connection %v.", err)
	}
	defer dc.CloseDs()
	// initialize our jwt authentication
	var u *auth.User
	if *i != "" {
		if err = u.BootstrapUsers(i, dc); err != nil {
			log.Fatalf("cannot bootstrap user %v", err)
		}
	}
	authMiddleware := u.AuthMiddleware(cs.CouchDb.Bucket, dc)
	// router := gin.Default()
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	doc := router.Group("/doc")
	router.LoadHTMLGlob(*s + "/*.html")
	v1 := router.Group(fmt.Sprintf("%s", *r))
	{
		ag := v1.Group("/")
		ag.Use(authMiddleware.MiddlewareFunc())
		v1.POST("/login", authMiddleware.LoginHandler)
		ag.PUT("/user", userAdd)
		ag.DELETE("/user/:id", userDelete)
		ag.GET("/user/:id", userList)
		ag.GET("/users", userList)
		v1.GET("/nutrients/food/:id", nutrientFdcID)
		v1.GET("/nutrients/foods", nutrientFdcIDs)
		v1.GET("/food/:id", foodFdcID)
		v1.GET("/foods", foodFdcIds)
		v1.GET("/foods/browse", foodsBrowse)
		v1.GET("/foods/search", foodsSearchGet)
		v1.POST("/foods/search", foodsSearchPost)
		v1.GET("/foods/count/:doctype", countsGet)
		v1.GET("/dictionary/:type", dictionaryBrowse)
		v1.GET("/docs/:type", specDoc)
		v1.POST("/nutrients/report", nutrientReportPost)
	}
	doc.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "apiDoc.html", nil)
	})
	endless.ListenAndServe(":"+*p, router)
}
