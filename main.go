// main
package main

import (

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
	r.Progress = make(chan string, 10)
	r.Result = make(chan MosResult, 1)
		
	return r
}

type MosProgress struct {
	
	Percent int
}

type MosResult struct {
	
	Mosaic *bytes.Buffer
}

var MosRequests = make(map[string]*MosRequest)
var MosQueue = make(chan *MosRequest, 100)
var nextid int = 0
var once sync.Once


func listen(w http.ResponseWriter, req *http.Request) {
	
	key := req.URL.Query().Get("key")
	fmt.Println("WTFWTFWTF", key)
	
	h, _ := w.(http.Hijacker)
	conn, rw, _ := h.Hijack()
	defer conn.Close()
	rw.Write([]byte("HTTP/1.1 200 OK\r\n"))
	rw.Write([]byte("Content-Type: text/event-stream\r\n\r\n"))
	rw.Flush()
	
	//w.Header().Set("Content-Type","text/event-stream")
	//io.WriteString(w,"\n\n")
	
	var mr *MosRequest// = MosRequests[key]
	var ok bool
	
	if mr, ok = MosRequests[key]; !ok {
		fmt.Println("key not found")
    return
	}
	
	for {
		
		select{
			case msg := <-mr.Progress:
			fmt.Println("goto ne: ", msg)
			rw.Write([]byte("data: " + msg+"\n\n"))
			rw.Flush()
			
			case result := <-mr.Result:
			fmt.Println("loks done")
			buf:=result.Mosaic
			rw.Write([]byte("data: "))
			rw.Write(buf.Bytes())
			rw.Write([]byte("\n\n"))
			rw.Flush()
			
			//io.WriteString(w, "data: " + strconv.Itoa(per)+"\n\n")
			
		}
	
		
	}
	
	
	
}

func hello(w http.ResponseWriter, req *http.Request) {
	
	mr := MosRequests["test"]
	mr.Progress <- "hey"
	nextid++
	
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
       // bytes[i] = byte(randInt(65,90))
    }
    return string(bytes)
}

func randInt(min int, max int) int {
    return min + rand.Intn(max-min)
		
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
	dict:=buildDictionary()

	count:=0
	
	for j:=0;j<out.Bounds().Max.Y;j++ {
		for i:=0;i<out.Bounds().Max.X;i++ {
			count++
			
			pixel := out.RGBAAt(i,j)
						
			var min float64 =999999
			var img *image.RGBA
			var mini int = -1
			
			for v := range dict {
				
				mi := dict[v]
				dif:=colorDistance(mi.AvgColor, &pixel)
					
				if dif<min {
					mini = v
					min = dif
				}
				
			}
			img = dict[mini].Image
			
					
			draw.Draw(mosaic, image.Rect(64*i,64*j,64*i+64,64*j+64), img, img.Bounds().Min, draw.Src)
			
			
		}
	}
	
//	var b bytes.Buffer
	
	
	
	mf,_ := os.OpenFile(mr.Key+".png",os.O_CREATE, 0666)
	defer mf.Close()
	png.Encode(mf, mosaic)
	
	mr.Progress<-"ready for download"
	//mr.Result <- MosResult{Mosaic:&b}
	
}


func main(){
	
	
  go func() {
		fmt.Println("wtf?")
			
	  for {
	    var mr *MosRequest;
      select {
    	case mr = <-MosQueue:
       
			fmt.Println("terms")
			for _,t := range mr.Terms {
				
				fmt.Println("terms: ", t)
				
			}
			
			buildMosaic(mr)
			
       fmt.Println("saving image with delay...")
       time.Sleep(time.Millisecond*1000)
			 saveImage(mr.Image, "uploaded" + strconv.Itoa(mr.Id)+".png")

      }
	  }
 	}()
	
 
	http.HandleFunc("/postimage", postimage)
	http.HandleFunc("/listen", listen)
	http.HandleFunc("/hello", hello)
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.ListenAndServe(":555", nil)



}

func old(){
	fmt.Println("imported and not used: \"fmt\"")

	
	//flickrdownload() 

	
	
	reader, err := os.Open("bm.jpg")
	if err != nil {
	    log.Fatal(err)
	}
	defer reader.Close()
	
	m, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	
	rgba,err:=convertImage(m)
	if err!=nil {
		panic(err)
	}
	
	
	
	height:=rgba.Bounds().Max.Y
	width:=rgba.Bounds().Max.X

	
	
	out:=downsample(rgba, image.Rect(0,0,width/8,height/8))
	
	
	//	f,_ := os.OpenFile("downsample.png",os.O_CREATE, 0666)
	//png.Encode(f, out)


 


	mosaic := image.NewRGBA(image.Rect(0,0,width*8,height*8))
	dict:=buildDictionary()

	count:=0
	
	for j:=0;j<out.Bounds().Max.Y;j++ {
		for i:=0;i<out.Bounds().Max.X;i++ {
			count++
			
			//rlum:=out.RGBAAt(i,j).R
			
			pixel := out.RGBAAt(i,j)
			
			//rlum:=lum(&pixel)
			
			var min float64 =999999
			var img *image.RGBA
			var mini int = -1
			
			for v := range dict {
				
				mi := dict[v]
				dif:=colorDistance(mi.AvgColor, &pixel)
				//fmt.Println(dif)
				
				if dif<min {
					mini = v
					min = dif
				}
				
			}
			img = dict[mini].Image
			
			/*
			for l,m := range dict {	
				dif:= math.Abs(float64(l)-float64(rlum))
				if dif< min {
					min=dif
					img=m
				}			
			}
							
			if img==nil {
				fmt.Println("fck",rlum)
			}else{
			//dimg:= downsample(img,image.Rect(0,0,64,64))
			
			draw.Draw(mosaic, image.Rect(64*i,64*j,64*i+64,64*j+64), img, img.Bounds().Min, draw.Src)
			}
			*/
			
			draw.Draw(mosaic, image.Rect(64*i,64*j,64*i+64,64*j+64), img, img.Bounds().Min, draw.Src)
			
			
		}
	}
	
	
	
	mf,_ := os.OpenFile("mosaic.png",os.O_CREATE, 0666)
	png.Encode(mf, mosaic)
	
}