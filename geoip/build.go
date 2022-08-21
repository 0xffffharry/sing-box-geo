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
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func Build() ([]byte, error) {
	// 下载 geoip 文件 （https://github.com/v2fly/domain-list-community）
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
	allCodes := make([]string, 0)
	countryMap := make(map[string][]*net.IPNet)
	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "sing-geoip",
		IPVersion:               6,
		Languages:               []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN"},
		RecordSize:              32,
		Inserter:                inserter.ReplaceWith,
		DisableIPv4Aliasing:     true,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		return nil, err
	}
	//
	for _, GeoIPEntry := range GeoIPList.Entry {
		code := GeoIPEntry.CountryCode
		countryData := make([]*net.IPNet, 0)
		for _, CIDR := range GeoIPEntry.Cidr {
			_, IPNet, err := net.ParseCIDR(fmt.Sprintf("%s/%s", net.IP(CIDR.Ip).String(), strconv.Itoa(int(CIDR.Prefix))))
			if err != nil {
				continue
			}
			countryData = append(countryData, IPNet)
		}
		allCodes = append(allCodes, code)
		countryMap[code] = countryData
	}
	// gdut
	allCodes = append(allCodes, "GDUT")
	gdutData := make([]*net.IPNet, 0)
	_, IP1, _ := net.ParseCIDR("222.200.96.0/19")
	_, IP2, _ := net.ParseCIDR("202.116.128.0/19")
	_, IP3, _ := net.ParseCIDR("2001:da8:2018::/48")
	gdutData = append(gdutData, IP1, IP2, IP3)
	countryMap["GDUT"] = gdutData
	// cn-cernet
	allCodes = append(allCodes, "CNCERNET")
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
	countryMap["CNCERNET"] = cncernetData
	// 写入数据
	if len(allCodes) == 0 {
		allCodes = make([]string, 0, len(countryMap))
		for code := range countryMap {
			allCodes = append(allCodes, code)
		}
	}
	sort.Strings(allCodes)
	codeMap := make(map[string]bool)
	for _, code := range allCodes {
		codeMap[code] = true
	}
	for code, data := range countryMap {
		if codeMap[code] {
			for _, item := range data {
				err := writer.Insert(item, mmdbtype.String(code))
				if err != nil {
					return nil, err
				}
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
