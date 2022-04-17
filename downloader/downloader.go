package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const (
	userAgent = `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36`
)

// FileDownloader 文件下载器
type FileDownloader struct {
	// 待下载文件总大小
	fileSize int
	// 下载源链接
	url string
	// 下载完成文件名
	outputFileName string
	// 文件切片数，对应为下载线程
	totalPart int
	// 文件输出目录
	outputDir string
	// 已完成文件切片
	doneFilePart []filePart
	// 文件下载完成校验，例如md5, SHA-256等
	md5 string
}

// filePart 文件分片
type filePart struct {
	// 文件分片的序号
	Index int
	// 开始byte
	From int
	// 结束byte
	To int
	// http下载得到的文件分片内容
	Data []byte
}

// FileDownloader 工厂方法
func NewFileDownloader(url, outputFileName, outputDir string, totalPart int, md5 string) *FileDownloader {
	if outputDir == "" {
		// 获取当前工作目录
		wd, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		outputDir = wd
	}
	return &FileDownloader{
		fileSize:       0,
		url:            url,
		outputFileName: outputFileName,
		outputDir:      outputDir,
		totalPart:      totalPart,
		doneFilePart:   make([]filePart, totalPart),
		md5:            md5,
	}
}

//Run 开始下载任务
func (d *FileDownloader) Run() error {
	fileTotalSize, err := d.getHeaderInfo()
	if err != nil {
		return err
	}
	d.fileSize = fileTotalSize

	jobs := make([]filePart, d.totalPart)
	eachSize := fileTotalSize / d.totalPart

	for i := range jobs {
		jobs[i].Index = i
		if i == 0 {
			jobs[i].From = 0
		} else {
			jobs[i].From = jobs[i-1].To + 1
		}
		if i < d.totalPart-1 {
			jobs[i].To = jobs[i].From + eachSize
		} else {
			// 最后一个filePart
			jobs[i].To = fileTotalSize - 1
		}
	}

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(job filePart) {
			defer wg.Done()
			err := d.downloadPart(job)
			if err != nil {
				log.Println("下载文件失败:", err, job)
			}
		}(j)
	}
	wg.Wait()

	return d.mergeFileParts()
}

/*
	head 获取要下载的文件的响应头(header)基本信息

	使用HTTP Method Head方法
*/
func (d *FileDownloader) getHeaderInfo() (int, error) {
	headers := map[string]string{
		"User-Agent": userAgent,
	}
	r, err := getNewRequest(d.url, "HEAD", headers)
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode > 299 {
		return 0, errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}

	// 检查是否支持断点续传
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Ranges
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return 0, errors.New("服务器不支持文件断点续传")
	}

	// 支持文件断点续传时，获取文件大小，名称等信息
	outputFileName, err := parseFileInfo(resp)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("get file info err: %v", err))
	}
	if d.outputFileName == "" {
		d.outputFileName = outputFileName
	}

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Length
	return strconv.Atoi(resp.Header.Get("Content-Length"))
}

// 下载分片
func (d *FileDownloader) downloadPart(c filePart) error {
	headers := map[string]string{
		"User-Agent": userAgent,
		"Range":      fmt.Sprintf("bytes=%v-%v", c.From, c.To),
	}
	r, err := getNewRequest(d.url, "GET", headers)
	if err != nil {
		return err
	}

	log.Printf("开始[%d]下载from:%d to:%d\n", c.Index, c.From, c.To)
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("服务器错误状态码: %v", resp.StatusCode))
	}
	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(bs) != (c.To - c.From + 1) {
		return errors.New("下载文件分片长度错误")
	}

	c.Data = bs
	d.doneFilePart[c.Index] = c
	return nil
}

// mergeFileParts 合并下载的文件
func (d *FileDownloader) mergeFileParts() error {
	path := filepath.Join(d.outputDir, d.outputFileName)

	log.Println("开始合并文件")
	mergedFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer mergedFile.Close()

	fileMd5 := sha256.New()
	totalSize := 0
	for _, s := range d.doneFilePart {
		_, err := mergedFile.Write(s.Data)
		if err != nil {
			fmt.Printf("error when merge file: %v\n", err)
		}
		fileMd5.Write(s.Data)
		totalSize += len(s.Data)
	}
	if totalSize != d.fileSize {
		return errors.New("文件不完整")
	}

	if d.md5 != "" {
		if hex.EncodeToString(fileMd5.Sum(nil)) != d.md5 {
			return errors.New("文件损坏")
		} else {
			log.Println("文件SHA-256校验成功")
		}
	}

	return nil
}

func getNewRequest(url, method string, headers map[string]string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}

	return r, nil
}

func parseFileInfo(resp *http.Response) (string, error) {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			return "", err
		}
		return params["filename"], nil
	}

	filename := filepath.Base(resp.Request.URL.Path)
	return filename, nil
}
