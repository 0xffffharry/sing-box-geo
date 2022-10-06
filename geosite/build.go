package geosite

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/sagernet/sing-box/common/geosite"
	"github.com/sagernet/sing/common"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func Build(geositeFile *os.File) error {
	V2rayDatData, err := downloadV2rayDat()
	if err != nil {
		return err
	}
	log.Println("geosite: download V2rayDatData success")
	// write geosite
	geosite_data, err := buildGeoSite(
		func() (map[string][]geosite.Item, error) {
			return parseV2rayDat(V2rayDatData), nil
		},
		personalRules,
	)
	if err != nil {
		return err
	}
	_, err = geositeFile.Write(geosite_data)
	if err != nil {
		return err
	}
	log.Println("geosite: geosite write success")
	return nil
}

type AddFunc func() (map[string][]geosite.Item, error)

func buildGeoSite(addFunc ...AddFunc) ([]byte, error) {
	if addFunc == nil || len(addFunc) == 0 {
		return nil, errors.New("no add func")
	}
	domainMap := make(map[string][]geosite.Item)
	log.Println("geosite: add func start")
	for _, F := range addFunc {
		d, err := F()
		if err != nil {
			log.Println(fmt.Sprintf("geosite: add func error: %s , continue...", err))
			continue
		}
		for k, v := range d {
			if _, ok := domainMap[k]; ok {
				domainMap[k] = append(domainMap[k], v...)
			} else {
				domainMap[k] = v
			}
		}
	}
	log.Println("geosite: add func success")
	output := bytes.NewBuffer(nil)
	err := geosite.Write(output, domainMap)
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func downloadV2rayDat() (*routercommon.GeoSiteList, error) {
	// 下载 geosite 文件 （https://github.com/Loyalsoldier/v2ray-rules-dat）
	geosite_dat_resp, err := http.Get("https://github.com/Loyalsoldier/v2ray-rules-dat/blob/release/geosite.dat?raw=true")
	if err != nil {
		return nil, err
	}
	geosite_dat, err := ioutil.ReadAll(geosite_dat_resp.Body)
	defer geosite_dat_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// 下载 geosite 文件 sha256sum
	geosite_dat_sha256sum_resp, err := http.Get("https://github.com/Loyalsoldier/v2ray-rules-dat/blob/release/geosite.dat.sha256sum?raw=true")
	if err != nil {
		return nil, err
	}
	geosite_dat_sha256sum, err := ioutil.ReadAll(geosite_dat_sha256sum_resp.Body)
	defer geosite_dat_sha256sum_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// 检查 geosite 文件完整
	checksha256sum := sha256.Sum256(geosite_dat)
	if hex.EncodeToString(checksha256sum[:]) != string(geosite_dat_sha256sum[:64]) {
		return nil, errors.New("geosite checksum mismatch")
	}
	// 还原构造
	GeoSiteList := routercommon.GeoSiteList{}
	err = proto.Unmarshal(geosite_dat, &GeoSiteList)
	if err != nil {
		return nil, err
	}
	return &GeoSiteList, nil
}

func parseV2rayDat(list *routercommon.GeoSiteList) map[string][]geosite.Item {
	domainMap := make(map[string][]geosite.Item)
	for _, GeoSiteEntry := range list.Entry {
		code := strings.ToLower(GeoSiteEntry.CountryCode)
		domains := make([]geosite.Item, 0, len(GeoSiteEntry.Domain)*2)
		attributes := make(map[string][]*routercommon.Domain)
		for _, domain := range GeoSiteEntry.Domain {
			if len(domain.Attribute) > 0 {
				for _, attribute := range domain.Attribute {
					attributes[attribute.Key] = append(attributes[attribute.Key], domain)
				}
				continue
			}
			switch domain.Type {
			case routercommon.Domain_Plain:
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomainKeyword,
					Value: domain.Value,
				})
			case routercommon.Domain_Regex:
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomainRegex,
					Value: domain.Value,
				})
			case routercommon.Domain_RootDomain:
				if strings.Contains(domain.Value, ".") {
					domains = append(domains, geosite.Item{
						Type:  geosite.RuleTypeDomain,
						Value: domain.Value,
					})
				}
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomainSuffix,
					Value: "." + domain.Value,
				})
			case routercommon.Domain_Full:
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomain,
					Value: domain.Value,
				})
			}
		}
		domainMap[code] = common.Uniq(domains)
		for attribute, attributeEntries := range attributes {
			attributeDomains := make([]geosite.Item, 0, len(attributeEntries)*2)
			for _, domain := range attributeEntries {
				switch domain.Type {
				case routercommon.Domain_Plain:
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomainKeyword,
						Value: domain.Value,
					})
				case routercommon.Domain_Regex:
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomainRegex,
						Value: domain.Value,
					})
				case routercommon.Domain_RootDomain:
					if strings.Contains(domain.Value, ".") {
						attributeDomains = append(attributeDomains, geosite.Item{
							Type:  geosite.RuleTypeDomain,
							Value: domain.Value,
						})
					}
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomainSuffix,
						Value: "." + domain.Value,
					})
				case routercommon.Domain_Full:
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomain,
						Value: domain.Value,
					})
				}
			}
			domainMap[code+"@"+attribute] = common.Uniq(attributeDomains)
		}
	}
	return domainMap
}

func personalRules() (map[string][]geosite.Item, error) {
	domainMap := make(map[string][]geosite.Item)
	// gdut
	gdut_eduweb_resp, err := http.Get("https://raw.githubusercontent.com/yaotthaha/gdut-edu-rule/master/gdut-edu-rule-geosite-vpn.txt?raw=true")
	if err != nil {
		return nil, err
	}
	gdut_eduweb, err := ioutil.ReadAll(gdut_eduweb_resp.Body)
	defer gdut_eduweb_resp.Body.Close()
	if err != nil {
		return nil, err
	}
	gdut_eduweb_list := strings.Split(string(gdut_eduweb), "\n")
	gdut_eduweb_domains := make([]geosite.Item, 0)
	for _, v := range gdut_eduweb_list {
		V := strings.SplitN(v, ":", 2)
		switch V[0] {
		case "full":
			gdut_eduweb_domains = append(gdut_eduweb_domains, geosite.Item{
				Type:  geosite.RuleTypeDomain,
				Value: V[1],
			})
		default:
			gdut_eduweb_domains = append(gdut_eduweb_domains, geosite.Item{
				Type:  geosite.RuleTypeDomainSuffix,
				Value: "." + V[0],
			})
		}
	}
	domainMap["gdut-eduweb"] = common.Uniq(gdut_eduweb_domains)
	return domainMap, nil
}
