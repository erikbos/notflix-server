package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"path"

	"github.com/XS4ALL/curlyconf-go"
	"github.com/gorilla/mux"

	"github.com/miquels/notflix-server/collection"
	"github.com/miquels/notflix-server/database"
	"github.com/miquels/notflix-server/imageresize"
	"github.com/miquels/notflix-server/jellyfin"
	"github.com/miquels/notflix-server/notflix"
)

var configFile = "notflix-server.cfg"

type cfgMain struct {
	Listen      string
	Tls         bool
	TlsCert     string
	TlsKey      string
	Appdir      string
	Cachedir    string
	Dbdir       string
	Logfile     string
	Collections []collection.Collection `cc:"collection"`
	Jellyfin    struct {
		// Indicates if we should auto-register Jellyfin users
		AutoRegister bool
		// JPEG quality for posters
		ImageQualityPoster int
	}
}

var config = cfgMain{
	Listen:  "127.0.0.1:8060",
	Logfile: "stdout",
}

var resizer *imageresize.Resizer

func main() {
	log.Printf("Parsing config file")

	p, err := curlyconf.NewParser(configFile, curlyconf.ParserNL)
	if err == nil {
		err = p.Parse(&config)
	}
	if err != nil {
		fmt.Println(err.(*curlyconf.ParseError).LongError())
		return
	}

	log.Printf("Parsing flags")
	logfile := flag.String("logfile", config.Logfile,
		"Path of logfile. Use 'syslog' for syslog, 'stdout' "+
			"for standard output, or 'none' to disable logging.")
	flag.Parse()

	log.Printf("setting logfile")
	switch *logfile {
	case "syslog":
		logw, err := syslog.New(syslog.LOG_NOTICE, "notflix")
		if err != nil {
			log.Fatalf("error opening syslog: %v", err)
		}
		log.SetOutput(logw)
	case "none":
		log.SetOutput(io.Discard)
	case "stdout":
	default:
		f, err := os.OpenFile(*logfile,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	log.SetFlags(0)

	log.Printf("dbinit")
	database, err := database.New(&database.Options{
		Filename: path.Join(config.Dbdir, "tink-items.db"),
	})
	if err != nil {
		log.Fatalf("database.New: %s", err)
	}
	go database.AccessTokenRepo.BackgroundJobs()
	go database.PlayStateRepo.BackgroundJobs()

	collection := collection.New(&collection.Options{
		Collections: config.Collections,
		Db:          database,
	})

	resizer = imageresize.New(imageresize.Options{
		Cachedir: config.Cachedir,
	})
	// XXX FIXME
	// if config.cachedir != "" {
	// 	go cleanCache(*datadir, config.cachedir, time.Hour)
	// }

	log.Printf("building mux")

	r := mux.NewRouter()

	n := notflix.New(&notflix.Options{
		Collections:  collection,
		Db:           database,
		Imageresizer: resizer,
		Appdir:       config.Appdir,
	})
	n.RegisterHandlers(r)

	j := jellyfin.New(&jellyfin.Options{
		Collections:        collection,
		Db:                 database,
		Imageresizer:       resizer,
		AutoRegister:       config.Jellyfin.AutoRegister,
		ImageQualityPoster: config.Jellyfin.ImageQualityPoster,
	})
	j.RegisterHandlers(r)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(config.Appdir)))

	server := HttpLog(r)
	addr := config.Listen

	log.Printf("Initializing collections..")
	collection.Init()
	go collection.Background()

	if config.Tls {
		log.Printf("Serving HTTPS on %s", addr)
		log.Fatal(http.ListenAndServeTLS(addr, config.TlsCert,
			config.TlsKey, server))
	} else {
		log.Printf("Serving HTTP on %s", addr)
		log.Fatal(http.ListenAndServe(addr, server))
	}
}
