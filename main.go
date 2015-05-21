// main
package main

import (
	"runtime"
	"image/jpeg"
	"encoding/json"
	//"encoding/base64"

	"math/rand"
//	"io"
	"sync"
	"time"
	//"strconv"
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
	TILE_X = 8
	TILE_Y = 8
	TILE_X_RESOLUTION =2
	TILE_Y_RESOLUTION =2
	MOSAIC_SCALE=8
	
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
	Progress chan string
	Result chan MosResult
}

type ImageResponse struct{
	
		Image *image.Image
		Err error	
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
			
			rw.Write([]byte("event: progress\n"))
			rw.Write([]byte("data: " + msg+"\n\n"))
			rw.Flush()
			
			case result := <-mr.Result:
			
			//str := base64.StdEncoding.EncodeToString(result.Mosaic.Bytes())
			//json:= "{\"height\":" + strconv.Itoa(result.Height) + ",\"width\":" + strconv.Itoa(result.Width) + ",\"base64\":\""+str+ "\"}"
			bb,_:= json.Marshal(result.Mosaic.Bytes())		
			
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

func postimage(w http.ResponseWriter, req *http.Request) {
	
	
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
		
	mr := NewMosRequest(rgba,terms)
	mr.Key=randomString(16)
	
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
}

func worker(queue <-chan string, results chan<- ImageResponse) {
    for q := range queue {

			m,err := downloadanddecode(q)

      results <- ImageResponse{Image:m,Err:err}
    }
}

func buildMosaic(mr *MosRequest){

	var dx = TILE_X_RESOLUTION
	var dy = TILE_Y_RESOLUTION
	rgba:=mr.Image
	
	height:=rgba.Bounds().Max.Y
	width:=rgba.Bounds().Max.X
	
	outscalex:=TILE_X/TILE_X_RESOLUTION
	outscaley:=TILE_Y/TILE_Y_RESOLUTION
	
	//out:=downsample(rgba, image.Rect(0,0,width/outscalex,height/outscaley))
	//mosaic := image.NewRGBA(image.Rect(0,0,width*MOSAIC_SCALE,height*MOSAIC_SCALE))
	
	//images:=flickrdownload(mr)	
	
	downx:=width/outscalex
	downy:=height/outscaley
	
	mosaicx:=downx*MOSAIC_SCALE*(TILE_X/TILE_X_RESOLUTION)
	mosaicy:=downy*MOSAIC_SCALE*(TILE_Y/TILE_Y_RESOLUTION)
	
	out:=downsample(rgba, image.Rect(0,0,downx,downy))
	mosaic := image.NewRGBA(image.Rect(0,0,mosaicx,mosaicy))
	
	mr.Progress<-"downloading source images"
	var urls []string = flickrSearch(500,mr.Terms...)
	
	dict:=downloadImages(urls)
	mr.Progress<-"building mosaic"
	
	const TILE_SCALE = TILE_Y*MOSAIC_SCALE
		
	for j:=0;j<out.Bounds().Max.Y;j+=dy {
		for i:=0;i<out.Bounds().Max.X;i+=dx {
						
			var min float64 =999999
			var img *image.RGBA
			var match int = -1
			var matches []int
			
			for v,mi := range dict {	
									
				if mi.Uses<4 || coinFlip(){				
					var dif float64 = 0
					
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
					
					if dif<120.0 {
						matches=append(matches,v)
					}				
				}				
			}
			
			if len(matches)>0 {
				match = matches[rand.Intn(len(matches))]
			}
			
			img = dict[match].Image
			dict[match].Uses++
							
			draw.Draw(mosaic, image.Rect(TILE_SCALE*i/TILE_X_RESOLUTION,TILE_SCALE*j/TILE_Y_RESOLUTION,TILE_SCALE*i/TILE_X_RESOLUTION+TILE_SCALE,TILE_SCALE*j/TILE_X_RESOLUTION+TILE_SCALE), img, img.Bounds().Min, draw.Src)
					
		}
	}
		
	var b bytes.Buffer		
	mr.Progress<-"downloading mosaic"
	
	jpeg.Encode(&b,mosaic,nil)
	//png.Encode(&b, mosaic)
	mr.Result <- MosResult{Mosaic:&b, Height:mosaic.Bounds().Max.Y, Width:mosaic.Bounds().Max.X}
}


func main(){
	
	runtime.GOMAXPROCS(4)
		
  go func() {
				
	  for {
	    //var mr *MosRequest;
      select {
    	case mr := <-MosQueue:
				buildMosaic(mr)
      }
	  }
 	}()
	
 
	http.HandleFunc("/postimage", postimage)
	http.HandleFunc("/listen", listen)
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.ListenAndServe(":555", nil)

}
