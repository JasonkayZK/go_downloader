## Go-Downloader

一个断点续传的例子。

### 使用方法

使用`NewFileDownloader`创建一个下载器，构造参数：

-   url：文件下载源地址；
-   outputFileName：输出文件名，若为空，则为原文件名；
-   outputDir：输出文件所在目录，若为空，则为当前工作目录；
-   totalPart：多少个文件分片，多少个分片就是多少个线程下载；
-   MD5：文件校验MD5，若为空，则不进行校验；

示例代码如下：

```go
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

```

### 更多说明

关于断点续传可见：

-   [Go实现HTTP断点续传多线程下载](https://jasonkayzk.github.io/2020/09/28/Go实现HTTP断点续传多线程下载/)

