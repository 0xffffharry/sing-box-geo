package main

import (
	"log"
	"os"
	"sing-box-geo/geoip"
	"sing-box-geo/geosite"
)

func main() {
	geositeBytes, err := geosite.Build()
	if err != nil {
		log.Fatalln(err)
	}
	geositeFile, err := os.Create(`geosite.db`)
	if err != nil {
		log.Fatalln(err)
	}
	defer geositeFile.Close()
	_, err = geositeFile.Write(geositeBytes)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("geosite build ok")
	//
	geoipBytes, err := geoip.Build(false)
	if err != nil {
		log.Fatalln(err)
	}
	geoipFile, err := os.Create(`geoip.db`)
	if err != nil {
		log.Fatalln(err)
	}
	defer geoipFile.Close()
	_, err = geoipFile.Write(geoipBytes)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("geoip build ok")
	//
	geoipMiniBytes, err := geoip.Build(true)
	if err != nil {
		log.Fatalln(err)
	}
	geoipMiniFile, err := os.Create(`geoip-mini.db`)
	if err != nil {
		log.Fatalln(err)
	}
	defer geoipMiniFile.Close()
	_, err = geoipMiniFile.Write(geoipMiniBytes)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("geoip mini build ok")
}
