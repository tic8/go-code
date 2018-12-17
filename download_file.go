package main

import (
	"fmt"
	"net/http"
	"time"
	"math"
	"path"
	"os"
	"log"
	"sync"
	"io"
)

const (
	DEFAULT_DOWNLOAD_BLOCK int64 = 4096
)

type Downloader struct {
	Url string
	count int
	DownloadBlock int64
	Filepath string
	ContentLength int64
	DownloadRange [][]int64
	File *os.File
	TempFiles []*os.File
	Wg sync.WaitGroup
}

func NewDownloader() *Downloader{
	dw := new(Downloader)
	dw.DownloadBlock = DEFAULT_DOWNLOAD_BLOCK
	return dw
}

func main(){

	dw := NewDownloader()

	time_start := time.Now()

	//url := "http://mirrors.163.com/centos/7.5.1804/isos/x86_64/CentOS-7-x86_64-Minimal-1804.iso"
	dw.Url = "http://mirrors.163.com/centos/7.5.1804/isos/x86_64/CentOS-7-x86_64-NetInstall-1804.torrent"

	req,err := http.NewRequest("HEAD",dw.Url,nil)
	if( err != nil ){
		log.Println(err)
		return
	}

	client := new(http.Client)
	resp,err := client.Do(req)
	if( err != nil ){
		log.Println(err)
		return
	}
	fmt.Println(resp)
	fmt.Println()

	if ( resp.Header.Get("Accept-Ranges") != "" ) {
		fmt.Println("支持range下载")
	}else{
		fmt.Println("不支持range下载")
	}

	dw.ContentLength = resp.ContentLength
	dw.count = int(math.Ceil(float64(resp.ContentLength / dw.DownloadBlock)))
	fmt.Println(dw.count)

	dw.Filepath = "/Users/yaya/Downloads/" + path.Base(dw.Url)

	fmt.Println(dw.Filepath)

	file,err := os.Create(dw.Filepath)
	if( err != nil ){
		log.Panicf("Create file %s error %v.\n", dw.Filepath, err)
	}
	defer file.Close()

	var range_start int64 = 0
	for i:= 0;i < dw.count; i++ {
		if (i != dw.count -1) {
			dw.DownloadRange = append(dw.DownloadRange, []int64{range_start, range_start + dw.DownloadBlock-1})
		}else{
			dw.DownloadRange = append(dw.DownloadRange,[]int64{range_start,dw.ContentLength})
		}
		range_start += dw.DownloadBlock
	}

	fmt.Println(dw.DownloadRange)

	for i := 0; i < len(dw.DownloadRange); i++ {
		range_i := fmt.Sprintf("%d-%d", dw.DownloadRange[i][0], dw.DownloadRange[i][1])
		fmt.Println(range_i)
		
		temp_file, err := os.OpenFile(dw.Filepath+"."+range_i, os.O_RDONLY|os.O_APPEND, 0)
		if( err != nil ){
			temp_file,_ = os.Create(dw.Filepath + "." + range_i)
		}else{
			fi, err := temp_file.Stat()
			if err == nil {
				dw.DownloadRange[i][0] += fi.Size()
			}
		}
		dw.TempFiles = append(dw.TempFiles,temp_file)
		

		
	}

	for i,_ := range dw.DownloadRange{
		dw.Wg.Add(1)
		go dw.Download(i)
	}

	dw.Wg.Wait()

	
	for i := 0; i < len(dw.TempFiles); i++{
		
		temp_file,_ := os.Open(dw.TempFiles[i].Name())
		cnt, err := io.Copy(file, temp_file)
		if cnt <= 0 || err != nil {
			log.Printf("Download #%d error %v.\n", i, err)
		}
		temp_file.Close()
		
	}


	file.Close()

	log.Printf("Download complete and store file %s.\n", dw.Filepath)

	defer func(){
		for i := 0; i < len(dw.TempFiles); i++ {
			err := os.Remove(dw.TempFiles[i].Name())
			if err != nil {
				log.Printf("Remove temp file %s error %v.\n", dw.TempFiles[i].Name(), err)
			}
		}
	}()


	time_end := time.Now()
	exec_time := time_end.Sub(time_start)
	fmt.Println("Total exec time:",exec_time)

}

func (dw *Downloader) Download(i int){
	defer dw.Wg.Done()
	fmt.Printf("----%v\n",i)
	range_i := fmt.Sprintf("%d-%d",dw.DownloadRange[i][0],dw.DownloadRange[i][1])
	fmt.Printf("Download #%d bytes %s.\n",i,range_i)

	defer dw.TempFiles[i].Close()

	req,err := http.NewRequest("GET",dw.Url,nil)
	req.Header.Set("Range","bytes="+range_i)
	client := new(http.Client)
	resp,err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Printf("Download #%d error %v\n",i,err)
	}else{
		io.Copy(dw.TempFiles[i],resp.Body)
		
		log.Printf("Download #%d complete.\n",i)
		/*
		if cnt == int64(dw.DownloadRange[i][1]-dw.DownloadRange[i][0]+1){
			log.Printf("Download #%d complete.\n",i)
		}
		*/

	}


}