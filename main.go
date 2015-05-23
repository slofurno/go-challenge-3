// main
package main

import (
	"io"
	"strconv"
	"runtime"
	"image/jpeg"
	"encoding/json"
	"math/rand"
	"time"
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

const (
	tileWidth = 8
	tileHeight = 8
	tileXResolution =2
	tileYResolution =2
	mosaicScale=8
	maxColorDifference = 120.0
	
)

type mosRequest struct {
	
	Image *image.RGBA
	Key string
	Id int
	Terms []string
	Progress chan string
	Result chan *image.RGBA
	Save bool
	Start time.Time
	End time.Time
}

type imageResponse struct{
	
		Image image.Image
		Err error	
}

func newMosRequest(img *image.RGBA, terms []string, tosave bool) *mosRequest {
	
	r:=&mosRequest{}
	r.Image=img
	r.Terms=terms
	r.Progress = make(chan string, 15)
	r.Result = make(chan *image.RGBA, 1)
	r.Save=tosave
	
	return r
}


var mosRequests = make(map[string]*mosRequest)
var mosQueue = make(chan *mosRequest, 100)
var savedMosaics []string

func getImages(w http.ResponseWriter, req *http.Request) {
	
	r,_:=json.Marshal(savedMosaics)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write([]byte(r))
}

func listen(w http.ResponseWriter, req *http.Request) {
	
	key := req.URL.Query().Get("key")
		
	h, _ := w.(http.Hijacker)
	conn, rw, _ := h.Hijack()
	defer conn.Close()
	
	rw.Write([]byte("HTTP/1.1 200 OK\r\n"))
	rw.Write([]byte("Content-Type: text/event-stream\r\n\r\n"))
	rw.Flush()
	
	var mr *mosRequest
	var ok bool
	
	if mr, ok = mosRequests[key]; !ok {
		fmt.Println("key not found")
    return
	}
	delete(mosRequests,key)
		
	disconnect:=make(chan bool, 1)
	
	go func(){
		_,err := rw.ReadByte()
		if err==io.EOF {
			disconnect<-true
		}
	}()
	
	for {
		
		select{
		case <-disconnect:
			fmt.Println("disconnected")
			return
			
		case msg := <-mr.Progress:
			
			rw.Write([]byte("event: progress\n"))
			rw.Write([]byte("data: " + msg+"\n\n"))
			rw.Flush()
			
		case mosaic := <-mr.Result:
			
			var b bytes.Buffer	
	
			jpeg.Encode(&b,mosaic,nil)
			
			//str := base64.StdEncoding.EncodeToString(result.Mosaic.Bytes())
			//json:= "{\"height\":" + strconv.Itoa(result.Height) + ",\"width\":" + strconv.Itoa(result.Width) + ",\"base64\":\""+str+ "\"}"
			bb,_:= json.Marshal(b.Bytes())		
			
			rw.Write([]byte("event: image\n"))
			rw.Write([]byte("data: "))
			rw.Write(bb)
			//rw.Write([]byte(json))
			rw.Write([]byte("\n\n"))
			rw.Flush()
			return	
			
		}
		
	}
		
}

func postImage(w http.ResponseWriter, req *http.Request) {
	
	
	defer req.Body.Close()
	buf, err := ioutil.ReadAll(req.Body)
	buffer:=bytes.NewReader(buf)

	m, _, err := image.Decode(buffer)
	
	if err != nil {
		log.Fatal(err)
		return
	}
	
	rgba := convertImage(m)
	
	qs:=req.URL.Query()
	terms:= qs["terms"]
	tosave:= false
	tosave,err=strconv.ParseBool(qs.Get("save"))
		
	mr := newMosRequest(rgba,terms,tosave)
	mr.Key=randomString(16)
	
	mosRequests[mr.Key] = mr
	
	mosQueue <- mr
	mr.Progress<-"mosaic queued"

	w.Write([]byte(mr.Key))
	

}

func saveJPG(img *image.RGBA, fn string) {
	
	f,err := os.OpenFile(fn,os.O_CREATE|os.O_RDWR, 0666)
	
	
	defer func() { 
    if err := f.Close(); err != nil {
         fmt.Println(err)
    }
	}()
	
	if err != nil{
		fmt.Println(err)
		return
	}
	
	err = jpeg.Encode(f, img,nil)
		if err != nil{
		fmt.Println(err)
		return
	}
	
}

func saveImage(img *image.RGBA, fn string) {
	
	f,err := os.OpenFile(fn,os.O_CREATE|os.O_RDWR, 0666)
	
	
	defer func() { 
    if err := f.Close(); err != nil {
         fmt.Println(err)
    }
	}()
	
	if err != nil{
		fmt.Println(err)
		return
	}
	
	err = png.Encode(f, img)
		if err != nil{
		fmt.Println(err)
		return
	}
	
}

func coinFlip() bool {
	
	if rand.Float64()>=.5 {
		return true
	}
	return false
	
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
			
	f,err := os.Open("static/images")//File("static/images",os.O_CREATE, 0666)
	defer f.Close()
	
	if err!=nil {
		return
	}
	
	fi,err:=f.Readdir(200)
	
	if err!=nil {
		return
	}
	
	for _,file:=range fi{
		savedMosaics=append(savedMosaics,file.Name())
	}
	
}

func worker(queue <-chan string, results chan<- imageResponse) {
    for q := range queue {

			m,err := downloadAndDecode(q)

      results <- imageResponse{Image:m,Err:err}
    }
}

func fitMosaic(rgba *image.RGBA, tiles []mosImage) *image.RGBA {
	
	var dx = tileXResolution
	var dy = tileYResolution
	
	height:=rgba.Bounds().Max.Y
	width:=rgba.Bounds().Max.X
	
	outscalex:=tileWidth/tileXResolution
	outscaley:=tileHeight/tileYResolution
	
	downx:=width/outscalex
	downy:=height/outscaley
	
	mosaicx:=downx*mosaicScale*(tileWidth/tileXResolution)
	mosaicy:=downy*mosaicScale*(tileHeight/tileYResolution)
	
	out:=downsample(rgba, image.Rect(0,0,downx,downy))
	mosaic := image.NewRGBA(image.Rect(0,0,mosaicx,mosaicy))
	
	const tileScale = tileHeight*mosaicScale
		
	for j:=0;j<out.Bounds().Max.Y;j+=dy {
		for i:=0;i<out.Bounds().Max.X;i+=dx {
						
			var min float64 =999999
			var img *image.RGBA
			match:= -1
			var matches []int
			
			for v,mi := range tiles {	
									
				if mi.Uses<4 || coinFlip(){				
					var dif float64
					
					for dj:=0;dj<dy;dj++ {
						for di:=0;di<dx;di++ {							
							pixel := out.RGBAAt(i+di,j+dj)
							tpixel := mi.Tile.RGBAAt(di,dj)
							dif+= colorDistance(&tpixel, &pixel)							
						}
					}			
											
					if dif<min {
						match = v
						min = dif
					}
					
					if dif<maxColorDifference {
						matches=append(matches,v)
					}				
				}				
			}
			
			if len(matches)>0 {
				match = matches[rand.Intn(len(matches))]
			}
			
			img = tiles[match].Image
			tiles[match].Uses++
							
			draw.Draw(mosaic, image.Rect(tileScale*i/tileXResolution,tileScale*j/tileYResolution,tileScale*i/tileXResolution+tileScale,tileScale*j/tileXResolution+tileScale), img, img.Bounds().Min, draw.Src)
					
		}
	}
	
	
	return mosaic
	/*	
	var b bytes.Buffer	
	
	jpeg.Encode(&b,mosaic,nil)
	return MosResult{Mosaic:&b, Height:mosaic.Bounds().Max.Y, Width:mosaic.Bounds().Max.X}
	*/
}

func buildMosaic(mr *mosRequest) *image.RGBA{
	
	mr.Progress<-"downloading source images"
	urls:= flickrSearch(500,mr.Terms...)
	
	tiles:=downloadImages(urls)
	mr.Progress<-"building mosaic"
	
	
	return fitMosaic(mr.Image,tiles)
	
	
	
	
}

func main(){
	
	runtime.GOMAXPROCS(4)
		
  go func() {
				
	  for {
	    //var mr *MosRequest;
      select {
    	case mr := <-mosQueue:
			
				mr.Start = time.Now()
				mosaic:=buildMosaic(mr)
				mr.End= time.Now()
				
				fmt.Println("elapsed time: ", mr.End.Sub(mr.Start))
				mr.Progress<-"downloading mosaic"
				
				if mr.Save {
					fmt.Println("saving mosaic...")
					saveJPG(mosaic,"static/images/"+mr.Key+".jpg")
					savedMosaics=append(savedMosaics,mr.Key+".jpg")
					thumb:=downsample(mosaic,image.Rect(0,0,300,300))
					saveJPG(thumb,"static/thumbs/"+mr.Key+".jpg")
				}
				mr.Result<-mosaic
      }
	  }
 	}()
	
 
	http.HandleFunc("/postimage", postImage)
	http.HandleFunc("/listen", listen)
	http.HandleFunc("/api/images",getImages)
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.ListenAndServe(":555", nil)

}
