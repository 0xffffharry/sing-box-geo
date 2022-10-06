package geoip

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

func Build(geoipFile *os.File) error {
	V2rayDatData, err := downloadV2rayDat()
	if err != nil {
		return err
	}
	log.Println("geoip: download V2rayDatData success")
	// write geoip
	geoip_data, err := buildGeoIP(
		func() (map[string][]*net.IPNet, error) {
			return parseV2rayDat(V2rayDatData), nil
		},
		personalRules,
	)
	if err != nil {
		return err
	}
	_, err = geoipFile.Write(geoip_data)
	if err != nil {
		return err
	}
	log.Println("geoip: geoip write success")
	return nil
}

type AddFunc func() (map[string][]*net.IPNet, error)

func buildGeoIP(addFunc ...AddFunc) ([]byte, error) {
	if addFunc == nil || len(addFunc) == 0 {
		return nil, errors.New("no add func")
	}
	countryMap := make(map[string][]*net.IPNet)
	log.Println("geoip: add rules start")
	for _, F := range addFunc {
		dataMap, err := F()
		if err != nil {
			log.Println(fmt.Sprintf("geoip: add func error: %s , continue...", err))
			continue
		}
		for code, data := range dataMap {
			codeLower := strings.ToLower(code)
			if _, ok := countryMap[codeLower]; !ok {
				countryMap[codeLower] = data
			} else {
				countryMap[codeLower] = append(countryMap[codeLower], data...)
			}
		}
	}
	log.Println("geoip: add rules success")
	allCodes := make([]string, 0)
	for code := range countryMap {
		allCodes = append(allCodes, code)
	}
	sort.Strings(allCodes)
	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "sing-geoip",
		Languages:               allCodes,
		IPVersion:               6,
		RecordSize:              24,
		Inserter:                inserter.ReplaceWith,
		DisableIPv4Aliasing:     true,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		return nil, err
	}
	for code, data := range countryMap {
		for _, item := range data {
			err := writer.Insert(item, mmdbtype.String(code))
			if err != nil {
				return nil, err
			}
		}
	}
	output := bytes.NewBuffer(nil)
	_, err = writer.WriteTo(output)
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func downloadV2rayDat() (*routercommon.GeoIPList, error) {
	// 下载 geoip 文件 （https://github.com/Loyalsoldier/v2ray-rules-dat）
	geoip_dat_resp, err := http.Get("https://github.com/Loyalsoldier/v2ray-rules-dat/blob/release/geoip.dat?raw=true")
	if err != nil {
		return nil, err
	}
	geoip_dat, err := ioutil.ReadAll(geoip_dat_resp.Body)
	defer geoip_dat_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// 下载 geoip 文件 sha256sum
	geoip_dat_sha256sum_resp, err := http.Get("https://github.com/Loyalsoldier/v2ray-rules-dat/blob/release/geoip.dat.sha256sum?raw=true")
	if err != nil {
		return nil, err
	}
	geoip_dat_sha256sum, err := ioutil.ReadAll(geoip_dat_sha256sum_resp.Body)
	defer geoip_dat_sha256sum_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// 检查 geoip 文件完整
	checksha256sum := sha256.Sum256(geoip_dat)
	if hex.EncodeToString(checksha256sum[:]) != string(geoip_dat_sha256sum[:64]) {
		return nil, errors.New("geoip checksum mismatch")
	}
	// 还原构造
	GeoIPList := routercommon.GeoIPList{}
	err = proto.Unmarshal(geoip_dat, &GeoIPList)
	if err != nil {
		return nil, err
	}
	return &GeoIPList, nil
}

func parseV2rayDat(list *routercommon.GeoIPList) map[string][]*net.IPNet {
	countryMap := make(map[string][]*net.IPNet)
	for _, GeoIPEntry := range list.Entry {
		code := GeoIPEntry.CountryCode
		countryData := make([]*net.IPNet, 0)
		for _, CIDR := range GeoIPEntry.Cidr {
			_, IPNet, err := net.ParseCIDR(fmt.Sprintf("%s/%s", net.IP(CIDR.Ip).String(), strconv.Itoa(int(CIDR.Prefix))))
			if err != nil {
				continue
			}
			countryData = append(countryData, IPNet)
		}
		countryMap[code] = countryData
	}
	return countryMap
}

func personalRules() (map[string][]*net.IPNet, error) {
	countryMap := make(map[string][]*net.IPNet)
	// gdut
	gdutData := make([]*net.IPNet, 0)
	_, IP1, _ := net.ParseCIDR("222.200.96.0/19")
	_, IP2, _ := net.ParseCIDR("202.116.128.0/19")
	_, IP3, _ := net.ParseCIDR("2001:da8:2018::/48")
	gdutData = append(gdutData, IP1, IP2, IP3)
	countryMap["gdut"] = gdutData
	// cn-cernet
	cncernet4_resp, err := http.Get("https://gaoyifan.github.io/china-operator-ip/cernet.txt")
	if err != nil {
		return nil, err
	}
	cncernet4, err := ioutil.ReadAll(cncernet4_resp.Body)
	defer cncernet4_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	cncernet4Data := make([]*net.IPNet, 0)
	for _, v := range strings.Split(string(cncernet4), "\n") {
		_, IPNet, err := net.ParseCIDR(v)
		if err != nil {
			continue
		}
		cncernet4Data = append(cncernet4Data, IPNet)
	}
	cncernet6_resp, err := http.Get("https://gaoyifan.github.io/china-operator-ip/cernet6.txt")
	if err != nil {
		return nil, err
	}
	cncernet6, err := ioutil.ReadAll(cncernet6_resp.Body)
	defer cncernet6_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	cncernet6Data := make([]*net.IPNet, 0)
	for _, v := range strings.Split(string(cncernet6), "\n") {
		_, IPNet, err := net.ParseCIDR(v)
		if err != nil {
			continue
		}
		cncernet6Data = append(cncernet6Data, IPNet)
	}
	cncernetData := append(cncernet4Data, cncernet6Data...)
	countryMap["cn-cernet"] = cncernetData
	return countryMap, nil
}
