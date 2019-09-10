package main

import (
	"bufio"
	"fmt"
	_ "github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/gocolly/colly"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func saveImg(link string, title string, name string, ch chan bool) {
	if res, err := http.Get(link); err == nil {
		ext := filepath.Base(link)
		p := path.Join("download", title, name+"_"+ext)
		if bin, err := ioutil.ReadAll(res.Body); err == nil {
			fmt.Println("保存 {{ " + link + " }} 到 {{ " + p + " }}")
			_ = ioutil.WriteFile(p, bin, 0777)
		}
	}
	ch <- true
}

func fchImg(title string, url string, ch chan bool) {
	decoder := mahonia.NewDecoder("gbk")
	cly := colly.NewCollector()
	cly.OnHTML("div.classBox.autoHeight", func(element *colly.HTMLElement) {
		// img src is inserted by script after page loaded
		script := decoder.ConvertString(element.DOM.Find("script[language=javascript]").Text())
		linkReg := regexp.MustCompile(`<IMG SRC='(?P<url>.*?)'>`)
		linkSubmatch := linkReg.FindAllStringSubmatch(script, -1)
		linkMatch := strings.ReplaceAll(linkSubmatch[0][1], `"+m2007+"`, "http://m8.1whour.com/")

		i1 := strings.LastIndex(url, "/")
		i2 := strings.LastIndex(url, ".")
		imgName := url[i1+1 : i2]
		go saveImg(linkMatch, title, imgName, ch)
	})
	_ = cly.Visit(url)
}

func fchEachEpisode(urlPath string, channel chan bool) {
	decoder := mahonia.NewDecoder("gbk")
	host := "http://m.ikkdm.com"
	link := host + urlPath
	cly := colly.NewCollector()
	cly.OnHTML("div.classBox.autoHeight", func(element *colly.HTMLElement) {

		txtA := decoder.ConvertString(element.DOM.Find("ul > center > li[class=txtA]").Text())
		split := strings.Split(txtA, "|")
		// this first is title
		title := strings.TrimSpace(split[0])
		// the second is episodeCount
		episodeCount, _ := strconv.ParseInt(strings.TrimSpace(strings.Split(split[1], "/")[1]), 10, 64)
		fmt.Printf("标题: %s, 页数: %d \n", title, episodeCount)

		if _, err := os.Stat("download/" + title); os.IsNotExist(err) {
			if err := os.Mkdir("download/"+title, 0777); err != nil {
				fmt.Println("无法为 {{ " + title + " }} 创建目录")
				return
			}
		}

		baseUrl := link[0 : strings.LastIndex(link, "/")+1]
		ch := make(chan bool, episodeCount)
		for i := int64(1); i <= episodeCount; i++ {
			url := baseUrl + strconv.FormatInt(i, 10) + ".htm"
			go fchImg(title, url, ch)
		}
		// 等待当前话的所有图片下载完成
		for i := int64(1); i <= episodeCount; i++ {
			<-ch
		}
		close(ch)
		channel <- true
	})
	_ = cly.Visit(link)
}

func handle(baseLink string, maxConnection int64) {
	var hrefs []string
	cly := colly.NewCollector()
	cly.ID = 1
	cly.OnHTML("#list", func(element *colly.HTMLElement) {
		// 获取所有话的链接
		element.ForEach("li > a[href]", func(i int, element *colly.HTMLElement) {
			href := element.Attr("href")
			hrefs = append(hrefs, href)
		})

		if maxConnection < 0 {
			maxConnection = int64(len(hrefs))
		} else if maxConnection > int64(len(hrefs)) {
			maxConnection = int64(len(hrefs))
		}
		fmt.Println("--------- 一共" + strconv.FormatInt(int64(len(hrefs)), 10) + "话 ---------")
		if _, err := os.Stat("download"); os.IsNotExist(err) {
			if err := os.Mkdir("download", 0777); err != nil {
				fmt.Println("创建下载目录失败")
				return
			}
		}
		fmt.Println("--------- 开始下载 ---------")
		channel := make(chan bool, maxConnection)
		connection := int64(0)
		for i := 0; i < len(hrefs); {
			if connection < maxConnection {
				connection++
				go fchEachEpisode(hrefs[i], channel)
				i++
			} else {
				<-channel
				connection--
			}
		}
		for connection > 0 {
			<-channel
			connection--
		}
		close(channel)
		fmt.Println("--------- 结束 ---------")
	})
	cly.OnResponse(func(response *colly.Response) {
		fmt.Println("收到" + baseLink + "的响应")
	})
	_ = cly.Visit(baseLink)
}

func main() {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("ikkdm的移动版链接 >>> ")
	baseLink, _ := reader.ReadString('\n')
	baseLink = baseLink[0 : len(baseLink)-1]

	fmt.Print("数据最大请求数（默认30，-1不限制）： ")
	max, _ := reader.ReadString('\n')
	maxConnection, err := strconv.ParseInt(max[:len(max)-1], 10, 64)
	if err != nil {
		maxConnection = 30
		fmt.Println("最大连接数：" + strconv.FormatInt(maxConnection, 10))
	}
	handle(baseLink, maxConnection)
}
