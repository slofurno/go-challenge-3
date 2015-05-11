package main

import (
	"sync"

	"strconv"
	"io/ioutil"
	"net/http"
	"image/png"
	"encoding/json"
	"io"
)

import(
	"image"
	_ "image/png"
	_ "image/jpeg"
	"os"
	"log"
	"image/color"
	
)


type FlickrPhoto struct{
	Id string `json:"id"`
	Owner string `json:"owner"`
	Secret string `json:"secret"`
	Server string `json:"server"`
	Farm int `json:"farm"`
	Title string `json:"title"`
	
}

func (p FlickrPhoto) downloadUrl() string {
	
	
	return "https://farm"+strconv.Itoa(p.Farm)+".staticflickr.com/"+p.Server+"/"+p.Id+"_"+p.Secret+".jpg"
	
}

type FlickrResponsePhotos struct{
	
	Page int `json:"page"`
	Pages int `json:"pages"`
	Perpage int `json:"perpage"`
	Total string `json:"total"`
	Photo []FlickrPhoto `json:"photo"`

	
}

type FlickrResponse struct{
	Photos FlickrResponsePhotos `json:"photos"`
	Stat string `json:"stat"`
	
}



func flickrdownload(){
	
	uri:="https://api.flickr.com/services/rest/?method=flickr.photos.search&api_key=749dec8d6d00d4df46215bf86e704bb0&text=fish&page=1&format=json&per_page=200&content_type=1"
	res, err:= http.Get(uri)
	contents, err := ioutil.ReadAll(res.Body)
	rawlen := len(contents)
	
	//flickr wraps our valid json response with jsonFlickrApi()
	j:=contents[14:rawlen-1]
	
	s:=string(j)
	
	log.Println(contents)
	log.Println(s)
  log.Println("wtf")
	
	var f FlickrResponse
	err=json.Unmarshal(j,&f)
	
	
	if err!=nil{
		log.Println(err)
	}
	
	log.Println("stat: " + f.Stat)
	plen := len(f.Photos.Photo)
	log.Println("length : " + strconv.Itoa(plen))
	
	var wg sync.WaitGroup
	
	for _, p := range f.Photos.Photo{
		
		wg.Add(1)
		go func(url string, filename string){
			defer wg.Done()
			
			downloadTest(url,filename)
		}(p.downloadUrl(),"p/"+p.Id+".jpg")
				
	}
	wg.Wait()
	/*
	decoder:=json.NewDecoder(res.Body)
	
	var f FlickrResponse
	err = decoder.Decode(&f)
	
	if err!=nil{
		log.Println(err)
	}
	*/
	
	//err := json.Unmarshal(data, &app)
	
	
	
}

func downloadTest(url string, fn string){
	
	out, err := os.Create(fn)
	if err!=nil {
		log.Println(err)
	}
	defer out.Close()
	res, err := http.Get(url)
	defer res.Body.Close()
	
	if err!=nil {
		log.Println(err)
	}
	
	n, err := io.Copy(out, res.Body)
	if err!=nil {
		log.Println(err)
	}
	log.Println("bytes downloaded : " + strconv.Itoa(int(n)))
}



func averageColor(img *image.RGBA, rect image.Rectangle) color.RGBA {
	
	r_avg := 0
	g_avg:=0
	b_avg:=0
	count:=0
	
	for i:=rect.Min.X; i<rect.Max.X;i++{
		for j:=rect.Min.Y;j<rect.Max.Y;j++{
			offset:=4*(j*img.Bounds().Max.X+i)
			
			r_avg+= int(img.Pix[offset])
			b_avg+= int(img.Pix[offset+1])
			g_avg+= int(img.Pix[offset+2])
			count++
			
		}
	}
		
	return color.RGBA{uint8(r_avg/count), uint8(g_avg/count), uint8(b_avg/count), 255}
	
}

func downsample(img *image.RGBA, size image.Rectangle) *image.RGBA {
	
	

	xratio := int(img.Bounds().Max.X/size.Max.X)
	yratio:=int(img.Bounds().Max.Y/size.Max.Y)
	
	out := image.NewRGBA(size)
	pixels := out.Pix
	
	for i:=0; i<size.Max.X;i++{
		for j:=0;j<size.Max.Y;j++{
			offset:=4*(j*size.Max.X+i)
			
			c:=averageColor(img, image.Rect(i*xratio, j*yratio, (i+1)*xratio, (j+1)*yratio))
			pixels[offset]=c.R
			pixels[offset+1]=c.B
			pixels[offset+2]=c.G
			pixels[offset+3]=255
			
		}
	}
	
	return out
	
}

func createGrayscale(img *image.RGBA) *image.RGBA {
	
	bounds := img.Bounds()
	width:=bounds.Max.X
	height:=bounds.Max.Y
  gray := image.NewRGBA(bounds)
	
	pixels := gray.Pix
	
	
	for i := 0; i < width;i++ {
		for j :=0;j<height;j++ {
			
			offset:=4*(width*j + i)
			
			r, g, b, _ := img.At(i, j).RGBA()
			lum:=0.299*float32(r) + 0.587*float32(g) + 0.114*float32(b)
			z:=uint8(int32(lum) >> 8)
			
			
			//fmt.Println(z)
			pixels[offset]=z
			pixels[offset+1]=z
			pixels[offset+2]=z
			pixels[offset+3]=255
			
		}
		
	}
	
	return gray
	
}

func openDirectory(fn string) []*image.RGBA {
	
	
	dir, _ := os.Open(fn)
	fi, _ := dir.Readdir(20)
	count:=len(fi)
	
	results := make([]*image.RGBA,count)
	
	for i:= 0; i<count;i++ {
		reader, _ := os.Open(fn + "/" + fi[i].Name())
		
		m, _, _ := image.Decode(reader)	
		rgba, _ := m.(*image.RGBA)
		reader.Close()
		results[i] = rgba
		
	}
	

	
	return results
	
}

func main(){
	
	flickrdownload()
	/*
	jpgs := openDirectory("p")
	
	for ii,j := range jpgs {
		
		gj := createGrayscale(j)
		gf,_ := os.OpenFile("g/" + strconv.Itoa(ii)+".png",os.O_CREATE, 0666)
		png.Encode(gf, gj)
	}
	*/
	
	reader, err := os.Open("front.png")
	if err != nil {
	    log.Fatal(err)
	}
	defer reader.Close()
	
	m, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	
	rgba, ok := m.(*image.RGBA)
	if !ok {
	       return
	}
	
  gray := createGrayscale(rgba)
	
	height:=gray.Bounds().Max.Y
	width:=gray.Bounds().Max.X
	
	
	//rgba, _ := img.(*image.RGBA)
	
	
	
	out:=downsample(gray, image.Rect(0,0,width/4,height/4))
	
	f,_ := os.OpenFile("small.png",os.O_CREATE, 0666)
	
	png.Encode(f, out)

	
}