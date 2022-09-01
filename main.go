package main

import (
	"log"
	"os"
	"sing-box-geo/geoip"
	"sing-box-geo/geosite"
)

func main() {
	geositeFile, err := os.Create(`geosite.db`)
	if err != nil {
		log.Fatalln(err)
	}
	defer geositeFile.Close()
	err = geosite.Build(geositeFile)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("geosite build ok")
	//
	geoipFile, err := os.Create(`geoip.db`)
	if err != nil {
		log.Fatalln(err)
	}
	defer geoipFile.Close()
	geoipMiniFile, err := os.Create(`geoip-mini.db`)
	if err != nil {
		log.Fatalln(err)
	}
	defer geoipMiniFile.Close()
	err = geoip.Build(geoipMiniFile, geoipFile)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("geoip build ok")
	log.Println("geoip mini build ok")
}
