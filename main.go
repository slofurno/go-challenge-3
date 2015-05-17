// main
package main

import (
	"encoding/base64"

	"math/rand"
//	"io"
	"sync"
	"time"
	"strconv"
	"bytes"
	"io/ioutil"
	"net/http"
	"image"
	"image/png"
	"image/draw"
	"log"
	"fmt"
	"os"
	_ "image/png"
	_ "image/jpeg"
)

type ImageDto struct {
	
	Data string
	Height int
	Width int
	
}

type MosRequest struct {
	
	Image *image.RGBA
	Key string
	Id int
	Terms []string
	Progress chan string//= make(chan MosProgress, 10)
	Result chan MosResult
}

func NewMosRequest(image *image.RGBA, terms []string) *MosRequest {
	
	r:=&MosRequest{}
	r.Image=image
	r.Terms=terms
	r.Progress = make(chan string, 15)
	r.Result = make(chan MosResult, 1)
		
	return r
}

type MosProgress struct {
	
	Percent int
}

type MosResult struct {
	
	Mosaic *bytes.Buffer
	Width int
	Height int
	
}

var MosRequests = make(map[string]*MosRequest)
var MosQueue = make(chan *MosRequest, 100)
var nextid int = 0
var once sync.Once


func listen(w http.ResponseWriter, req *http.Request) {
	
	key := req.URL.Query().Get("key")
		
	h, _ := w.(http.Hijacker)
	conn, rw, _ := h.Hijack()
	defer conn.Close()
	rw.Write([]byte("HTTP/1.1 200 OK\r\n"))
	rw.Write([]byte("Content-Type: text/event-stream\r\n\r\n"))
	rw.Flush()
	
	
	var mr *MosRequest
	var ok bool
	
	if mr, ok = MosRequests[key]; !ok {
		fmt.Println("key not found")
    return
	}else{
		delete(MosRequests,key)
	}
	
	for {
		
		select{
			case msg := <-mr.Progress:
			fmt.Println("goto ne: ", msg)
			rw.Write([]byte("event: progress\n"))
			rw.Write([]byte("data: " + msg+"\n\n"))
			rw.Flush()
			
			case result := <-mr.Result:
			
			str := base64.StdEncoding.EncodeToString(result.Mosaic.Bytes())
			
			json:= "{\"height\":" + strconv.Itoa(result.Height) + ",\"width\":" + strconv.Itoa(result.Width) + ",\"base64\":\""+str+ "\"}"
			
			rw.Write([]byte("event: image\n"))
			rw.Write([]byte("data: "))
			//rw.Write(buf.Bytes())
			rw.Write([]byte(json))
			rw.Write([]byte("\n\n"))
			rw.Flush()
			return	
			
		}
		
	}
		
}

func postimage(w http.ResponseWriter, req *http.Request) {
	
	
	
	buf, err := ioutil.ReadAll(req.Body)
	buffer:=bytes.NewReader(buf)

	m, _, err := image.Decode(buffer)
	
	if err != nil {
		log.Fatal(err)
	}
	
	rgba,err := convertImage(m)
	
	
	qs:=req.URL.Query()
	terms:= qs["terms"]
		
	mr := NewMosRequest(rgba,terms)
	mr.Key=randomString(16)
	//mr := MosRequest{Image:rgba, Id:nextid}
	nextid++
	
	MosRequests[mr.Key] = mr
	
	MosQueue <- mr
	mr.Progress<-"mosaic queued"

	w.Write([]byte(mr.Key))
	

}

func saveImage(img *image.RGBA, fn string) {
	
	f,_ := os.OpenFile(fn,os.O_CREATE, 0666)
	defer f.Close()
	png.Encode(f, img)
	
}

var alpha = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomString(l int ) string {
	 
    bytes := make([]byte, l)
    for i:=0 ; i<l ; i++ {
			bytes[i]= alpha[rand.Intn(len(alpha))]
    }
    return string(bytes)
}


func init(){
	 rand.Seed( time.Now().UTC().UnixNano())
}

func buildMosaic(mr *MosRequest){

	mr.Progress<-"starting mosaic"
	rgba:=mr.Image
	
	height:=rgba.Bounds().Max.Y
	width:=rgba.Bounds().Max.X
	
	out:=downsample(rgba, image.Rect(0,0,width/8,height/8))
	mosaic := image.NewRGBA(image.Rect(0,0,width*8,height*8))
	
	images:=flickrdownload(mr)	
	mr.Progress<-"processing images"
	dict:=buildDictionary(images)
	mr.Progress<-"building mosaic"
	
	for j:=0;j<out.Bounds().Max.Y;j++ {
		for i:=0;i<out.Bounds().Max.X;i++ {
			
			pixel := out.RGBAAt(i,j)
						
			var min float64 =999999
			var img *image.RGBA
			var match int = -1
			
			for v := range dict {
				
				mi := dict[v]
				//TODO:higher resolution color dif
				dif:=colorDistance(mi.AvgColor, &pixel)
					
				if dif<min {
					match = v
					min = dif
				}
				
			}
			img = dict[match].Image
							
			draw.Draw(mosaic, image.Rect(64*i,64*j,64*i+64,64*j+64), img, img.Bounds().Min, draw.Src)
					
		}
	}
	
	var b bytes.Buffer
	
		
	mr.Progress<-"downloading mosaic"
	
	png.Encode(&b, mosaic)
	mr.Result <- MosResult{Mosaic:&b, Height:mosaic.Bounds().Max.Y, Width:mosaic.Bounds().Max.X}
	
}


func main(){
		
  go func() {
				
	  for {
	    var mr *MosRequest;
      select {
    	case mr = <-MosQueue:
				buildMosaic(mr)
      }
	  }
 	}()
	
 
	http.HandleFunc("/postimage", postimage)
	http.HandleFunc("/listen", listen)
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.ListenAndServe(":555", nil)

}
