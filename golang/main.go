package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	_ "github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/gocolly/colly"
)

var (
	baseLocation = "download"
	decoder      = mahonia.NewDecoder("gbk")
	includes     = map[string]bool{}
	excludes     = map[string]bool{}
)

// Episode 每一话的标题和链接
type Episode struct {
	title string
	url   string
}

func saveImg(link string, title string, name string, ch chan bool) {
	if res, err := http.Get(link); err == nil {
		ext := filepath.Base(link)
		p := path.Join(baseLocation, title, name+"_"+ext)
		if bin, err := ioutil.ReadAll(res.Body); err == nil {
			_ = ioutil.WriteFile(p, bin, 0777)
		}
	}
	ch <- true
}

func fchImg(title string, url string, ch chan bool) {
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

func fchEachEpisode(episode Episode, channel chan bool) {
	url := "http://m.ikkdm.com" + episode.url
	cly := colly.NewCollector()
	cly.OnHTML("div.classBox.autoHeight", func(element *colly.HTMLElement) {

		lis := element.DOM.Find("div.bottom ul.subNav li")
		var info string
		lis.Each(func(i int, selection *goquery.Selection) {
			if 1 == i {
				info = selection.Text()
			}
		})
		split := strings.Split(info, "/")
		episodeCount, _ := strconv.ParseInt(strings.TrimSpace(split[1]), 10, 64)

		if _, err := os.Stat(filepath.Join(baseLocation, episode.title)); os.IsNotExist(err) {
			if err := os.Mkdir(filepath.Join(baseLocation, episode.title), 0777); err != nil {
				fmt.Println("无法为 {{ " + episode.title + " }} 创建目录")
				return
			}
		}

		baseURL := url[0 : strings.LastIndex(url, "/")+1]
		ch := make(chan bool, episodeCount)
		for i := int64(1); i <= episodeCount; i++ {
			url := baseURL + strconv.FormatInt(i, 10) + ".htm"
			go fchImg(episode.title, url, ch)
		}
		// 等待当前话的所有图片下载完成
		for i := int64(1); i <= episodeCount; i++ {
			<-ch
		}
		close(ch)
		fmt.Println("{{ " + episode.title + " }}下载完成")
		channel <- true
	})
	_ = cly.Visit(url)
}

func contains(collection map[string]bool, title string) bool {
	for k := range collection {
		if strings.HasPrefix(title, k) {
			return true
		}
	}
	return false
}

func handle(baseURL string, maxConnection int64) {
	var episodes []Episode
	cly := colly.NewCollector()
	cly.ID = 1
	cly.OnHTML("body", func(element *colly.HTMLElement) {
		// 获取所有话的链接
		baseLocation = filepath.Join(baseLocation, decoder.ConvertString(element.DOM.Find("#comicName").Text()))

		element.ForEach("#list > li > a[href]", func(i int, element *colly.HTMLElement) {
			href := element.Attr("href")
			title := decoder.ConvertString(element.Text)

			// 判断是否需要下载
			order := strings.Split(title, " ")[1]
			if (len(includes) == 0 || contains(includes, order)) && !contains(excludes, order) {
				episodes = append(episodes, Episode{title: title, url: href})
			}
		})

		if maxConnection < 0 || maxConnection > int64(len(episodes)) {
			maxConnection = int64(len(episodes))
		}
		fmt.Println("--------- 一共" + strconv.FormatInt(int64(len(episodes)), 10) + "话 ---------")
		if _, err := os.Stat(baseLocation); os.IsNotExist(err) {
			if err := os.MkdirAll(baseLocation, 0777); err != nil {
				fmt.Println("创建下载目录失败")
				return
			}
		}

		fmt.Println("--------- 开始下载 ---------")
		channel := make(chan bool, maxConnection)
		connection := int64(0)
		count := 0
		for i := 0; i < len(episodes); {
			if connection < maxConnection {
				connection++
				go fchEachEpisode(episodes[i], channel)
				i++
			} else {
				<-channel
				connection--
				count++
				fmt.Println("已保存" + strconv.Itoa(count) + "话")
			}
		}
		for connection > 0 {
			<-channel
			connection--
			count++
			fmt.Println("已保存" + strconv.Itoa(count) + "话")
		}
		close(channel)
		fmt.Println("--------- 结束 ---------")
	})
	cly.OnResponse(func(response *colly.Response) {
		fmt.Println("收到" + baseURL + "的响应")
	})
	_ = cly.Visit(baseURL)
}

func main() {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("ikkdm的移动版链接 >>> ")
	baseLink, _ := reader.ReadString('\n')
	baseLink = baseLink[0 : len(baseLink)-1]

	fmt.Print("数据最大请求数（默认10，-1不限制）： ")
	max, _ := reader.ReadString('\n')
	maxConnection, err := strconv.ParseInt(max[:len(max)-1], 10, 64)
	if err != nil {
		maxConnection = 10
		fmt.Println("最大连接数：" + strconv.FormatInt(maxConnection, 10))
	}

	// Includes
	fmt.Print("包括（设置包含项，空为全部）: ")
	if include, err := reader.ReadString('\n'); err == nil && include != "\n" {
		includeList := strings.Split(include[:len(include)-1], ",")
		for _, v := range includeList {
			includes[v] = true
		}
	}
	// Excludes
	fmt.Print("不包括（设置剔除项）：")
	if exclude, err := reader.ReadString('\n'); err == nil && exclude != "\n" {
		excludeList := strings.Split(exclude[:len(exclude)-1], ",")
		for _, v := range excludeList {
			excludes[v] = true
		}
	}
	handle(baseLink, maxConnection)
}
