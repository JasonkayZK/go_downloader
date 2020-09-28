package main

import (
	"fmt"
	"log"
	"time"

	. "github.com/jasonkayzk/go_downloader/downloader"
)

func main() {
	startTime := time.Now()
	url := "https://download.jetbrains.com/go/goland-2020.2.2.dmg"
	// SHA-256: https://download.jetbrains.com/go/goland-2020.2.2.dmg.sha256?_ga=2.223142619.1968990594.1597453229-1195436307.1493100134
	md5 := "3af4660ef22f805008e6773ac25f9edbc17c2014af18019b7374afbed63d4744"
	downloader := NewFileDownloader(url, "", "", 8, md5)
	if err := downloader.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n 文件下载完成耗时: %f second\n", time.Now().Sub(startTime).Seconds())
}
