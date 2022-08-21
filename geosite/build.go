package geosite

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/sagernet/sing-box/common/geosite"
	"github.com/sagernet/sing/common"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
	"strings"
)

func Build() ([]byte, error) {
	// 下载 geosite 文件 （https://github.com/v2fly/domain-list-community）
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
	// 类型转换
	domainMap := make(map[string][]geosite.Item)
	for _, GeoSiteEntry := range GeoSiteList.Entry {
		domains := make([]geosite.Item, 0, len(GeoSiteEntry.Domain)*2)
		for _, domain := range GeoSiteEntry.Domain {
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
		domainMap[strings.ToLower(GeoSiteEntry.CountryCode)] = common.Uniq(domains)
	}
	// 自定义规则
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
	domainMap["gduteduweb"] = common.Uniq(gdut_eduweb_domains)
	output := bytes.NewBuffer(nil)
	err = geosite.Write(output, domainMap)
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
