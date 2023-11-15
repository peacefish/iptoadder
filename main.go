package main

import (
	"encoding/json"
	"flag"
	"github.com/lionsoul2014/ip2region/binding/golang/ip2region"
	"github.com/thinkeridea/go-extend/exnet"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"io/ioutil"
	"fmt"
)

var (
	wg    = sync.WaitGroup{}
	port  = ""
	d     = "" // 下载标识
	dbUrl = map[string]string{
		"1": "https://fastly.jsdelivr.net/gh/lionsoul2014/ip2region@master/v1.0/data/ip2region.db",
		"2": "https://fastly.jsdelivr.net/gh/bqf9979/ip2region@master/data/ip2region.db",
	}
)

const (
	ipDbPath     = "./ip2region.db"
	defaultDbUrl = "2" // 默认下载 来自 bqf9979 仓库的 ip db文件
)

type JsonRes struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
type LocationResponse struct {
	Status     int    `json:"status"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	Result     Result `json:"result"`
}

type Result struct {
	IP       string   `json:"ip"`
	Location Location `json:"location"`
	ADInfo   ADInfo   `json:"ad_info"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type ADInfo struct {
	Nation      string `json:"nation"`
	Province    string `json:"province"`
	City        string `json:"city"`
	District    string `json:"district"`
	Adcode      int    `json:"adcode"`
	NationCode  int    `json:"nation_code"`
}
type IpInfo struct {
	Ip       string `json:"ip"`
	Country  string `json:"country"`  // 国家
	Province string `json:"province"` // 省
	City     string `json:"city"`     // 市
	County   string `json:"county"`   // 县、区
	Region   string `json:"region"`   // 区域位置
	ISP      string `json:"isp"`      // 互联网服务提供商
}

func init() {
	_p := flag.String("p", "9090", "本地监听的端口")
//	_d := flag.String("d", "0", "仅用于下载最新的ip地址库，保存在当前目录")
	flag.Parse()

	port = *_p
//	d = *_d

//	if d != "0" {
//		if value, ok := dbUrl[d]; ok {
//			downloadIpDb(value)
//		} else {
	//		downloadIpDb(dbUrl[defaultDbUrl])
//		}
//		os.Exit(1)
//	}

	checkIpDbIsExist()
}

func main() {
	http.HandleFunc("/", queryIp)

	link := "http://127.0.0.1:" + port

	log.Println("监听端口", link)
	listenErr := http.ListenAndServe(":"+port, nil)
	if listenErr != nil {
		log.Fatal("ListenAndServe: ", listenErr)
	}
}

func queryIp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/json")

	defer func() {
		//捕获 panic
		if err := recover(); err != nil {
			log.Println("查询ip发生错误", err)
		}
	}()

	if r.URL.Path != "/" {
		w.WriteHeader(404)
		msg, _ := json.Marshal(&JsonRes{Code: 4000, Msg: r.URL.Path + " 404 NOT FOUND !"})
		w.Write(msg)
		return
	}

	r.ParseForm() // 解析参数

	ip := r.FormValue("ip")

	if ip == "" {
		// 获取当前客户端 IP
		ip = getIp(r)
	}
	result, err := GetIpRegion(ip)
	if err == nil &&  result.Status==0 {
	  ipInfo := &IpInfo{
                Ip:       ip,
                ISP:      "",
                Country:  result.Result.ADInfo.Nation,
                Province: result.Result.ADInfo.Province,
                City:     result.Result.ADInfo.City,
                County:   result.Result.ADInfo.District,
                Region:   "",
        }       
        msg, _ := json.Marshal(JsonRes{Code: 200, Data: ipInfo})
        w.Write(msg)
        return
	}
	region, err := ip2region.New(ipDbPath)
	defer region.Close()

	if err != nil {
		msg, _ := json.Marshal(&JsonRes{Code: 4001, Msg: err.Error()})
		w.Write(msg)
		return
	}

	info, searchErr := region.MemorySearch(ip)

	if searchErr != nil {
		msg, _ := json.Marshal(JsonRes{Code: 4002, Msg: searchErr.Error()})
		w.Write(msg)
		return
	}
	
	// 赋值查询结果
	ipInfo := &IpInfo{
		Ip:       ip,
		ISP:      info.ISP,
		Country:  info.Country,
		Province: info.Province,
		City:     info.City,
		County:   "",
		Region:   info.Region,
	}
	msg, _ := json.Marshal(JsonRes{Code: 200, Data: ipInfo})
	w.Write(msg)
	return
}

// GetIpRegion 获取指定 IP 的区域信息
func GetIpRegion(ip string) (*LocationResponse, error) {
	apiURL := fmt.Sprintf("https://apis.map.qq.com/ws/location/v1/ip?key=腾讯地图key&ip=%s&output=json", ip)
	response, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var locationResponse LocationResponse
	err = json.Unmarshal(body, &locationResponse)
	if err != nil {
		return nil, err
	}

	return &locationResponse, nil
}

func getIp(r *http.Request) string {
	ip := exnet.ClientPublicIP(r)
	if ip == "" {
		ip = exnet.ClientIP(r)
	}
	return ip
}

func checkIpDbIsExist() {
	if d == "1" || d == "0" || d == "" {
		if _, err := os.Stat("./ip2region.db"); os.IsNotExist(err) {
			log.Println("ip 地址库文件不存在")
			downloadIpDb(dbUrl[defaultDbUrl])
		}
	}

	if d == "2" {
		if _, err := os.Stat(ipDbPath); os.IsNotExist(err) {
			log.Println("ip 地址库文件不存在")
			downloadIpDb(dbUrl[defaultDbUrl])
		}
	}

}

func downloadIpDb(url string) {
	log.Println("正在下载最新的 ip 地址库...：" + url)
	wg.Add(1)
	go func() {
		downloadFile(ipDbPath, url)
		wg.Done()
	}()
	wg.Wait()
	log.Println("下载完成")
}

// @link https://studygolang.com/articles/26441
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
