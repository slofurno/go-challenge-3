package main

import (
	"errors"
"reflect"
	"fmt"
	"strings"
//	"fmt"

  "io"
	"math"
	"sync"

	"strconv"
	"io/ioutil"
	"net/http"
	"image/png"
	"encoding/json"
	//"io"
)

import(
	"image"
	_ "image/png"
	_ "image/jpeg"
	"os"
	"log"
	"image/color"
	
)

type MosImage struct {
	
	Image *image.RGBA
	AvgColor *color.RGBA
}


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
	
	uri:="https://api.flickr.com/services/rest/?method=flickr.photos.search&api_key=749dec8d6d00d4df46215bf86e704bb0&text=corvette&page=1&format=json&per_page=700&content_type=1"
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
		}(p.downloadUrl(),"jpgs/"+p.Id+".jpg")
				
	}
	wg.Wait()
	
	
	//m = make(map[string]int)
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

func convertToPNG(w io.Writer, r io.Reader) error {
 img, _, err := image.Decode(r)
 if err != nil {
  return err
 }
 return png.Encode(w, img)
}

func downloadTest(url string, fn string){
	

		
		
	res, err := http.Get(url)
	defer res.Body.Close()
	
	if err!=nil {
		log.Println(err)
	}
	
	out, err := os.Create(fn)
	if err!=nil {
		log.Println(err)
	}
	defer out.Close()
		
	n, err := io.Copy(out, res.Body)
	if err!=nil {
		log.Println(err)
	}
	log.Println("bytes downloaded : " + strconv.Itoa(int(n)))
		
	
	
}



func averageColor(img *image.RGBA, rect image.Rectangle) color.RGBA {
	
	var r_sum float64 =0
	var g_sum float64 =0
	var b_sum float64 =0
	var count float64 =0
	
	pixels := img.Pix
	//fmt.Println("len",len(pixels))
	
	stride:=img.Bounds().Max.X
	
	for i:=rect.Min.X; i<rect.Max.X;i++{
		for j:=rect.Min.Y;j<rect.Max.Y;j++{
			offset:=4*(j*stride+i)
			
			
			r_sum+= sRGBtoLinear(pixels[offset])
			g_sum+= sRGBtoLinear(pixels[offset+1])
			b_sum+= sRGBtoLinear(pixels[offset+2])
			/*
			r_avg+= int(img.Pix[offset])
			b_avg+= int(img.Pix[offset+1])
			g_avg+= int(img.Pix[offset+2])
			*/
			count++
			
		}
	}
	
		
	return color.RGBA{lineartosRGB(r_sum/count), lineartosRGB(g_sum/count), lineartosRGB(b_sum/count), 255}
	
}

/*
	for i:=xoffset; i<xmax;i++{
		for j:=yoffset;j<ymax;j++{
			*/

func downsample(img *image.RGBA, size image.Rectangle) *image.RGBA {
	
	//xoffset:=int((img.Bounds().Max.X%size.Max.X)/2)
	//yoffset:=int((img.Bounds().Max.Y%size.Max.Y)/2)
	
//	xmax:=xoffset+size.Max.X
	//ymax:=yoffset+size.Max.Y
  
	xratio := int(img.Bounds().Max.X/size.Max.X)
	yratio:=int(img.Bounds().Max.Y/size.Max.Y)
	
	minratio:=xratio
	
	if yratio<xratio {
		minratio=yratio
	}
	
	xoffset:= int((img.Bounds().Max.X - size.Max.X*minratio)/2)
	yoffset:= int((img.Bounds().Max.Y - size.Max.Y*minratio)/2)
		
	fmt.Println(xoffset,yoffset,minratio*size.Max.X,minratio*size.Max.Y)
	
	out := image.NewRGBA(size)
	pixels := out.Pix
	
	for i:=0; i<size.Max.X;i++{
		for j:=0;j<size.Max.Y;j++{
			offset:=4*(j*size.Max.X+i)
			
			r:=image.Rect(i*minratio+xoffset, j*minratio+yoffset, (i+1)*minratio+xoffset, (j+1)*minratio+yoffset)
						
			c:=averageColor(img, r)
			
			//lum:= 0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B)
		//	fmt.Println("average lum : ", lum)
			
			pixels[offset]=c.R
			pixels[offset+1]=c.G
			pixels[offset+2]=c.B
			pixels[offset+3]=255
			
		}
	}
	
	return out
	
}

func lum(c *color.RGBA) float64 {
	return float64(0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B))
}

func averageLum(img *image.RGBA, r image.Rectangle) float32 {
	
	c:=averageColor(img, r)	
	lum:=0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B)
	
	return lum
	
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

func YCbCrToRGB(src *image.YCbCr) *image.RGBA {
	
	dst:=image.NewRGBA(src.Bounds())
	pix:=dst.Pix
	//YCbCrSubsampleRatio420blazeit
	
	fmt.Println("lens",len(src.Y), len(src.Cr),src.Bounds().String(),src.CStride, src.YStride, src.SubsampleRatio.String(), src.Opaque())
	
	
	c:=0
	for j:=0;j<src.Bounds().Max.Y;j++ {
		
		for i:=0;i<src.Bounds().Max.X;i++ {
			/*
			
			yi:=j*src.YStride+i
			ci:=int((j*src.CStride+i)/2)
						
			y:=float32(src.Y[yi])
			cb:=float32(src.Cb[ci])
			cr:=float32(src.Cr[ci])	
			
			r:=y + 1.402*(cr-128)
			g:=y -0.34414*(cb-128)-0.71414*(cr-128)
			b:=y+1.772*(cb-128)
			*/
			r1,g1,b1,_ := src.At(i,j).RGBA()
			
			pix[4*c]=uint8(r1)
			pix[4*c+1]=uint8(g1)
			pix[4*c+2]=uint8(b1)
			pix[4*c+3]=255
			
			c++
			
		}
		
	}
	
	return dst
	
	
}

func copyPixels(src *image.RGBA, r image.Rectangle) *image.RGBA {
	
	dr := image.Rect(0,0,r.Max.X-r.Min.X, r.Max.Y-r.Min.Y)
	
	dst := image.NewRGBA(dr)
	
	srcwidth := src.Bounds().Max.X
	//dstwidth := dst.Bounds().Max.X
	
	sp:=src.Pix
	dp:=dst.Pix
	
	di:=0
	
	for j:= r.Min.Y ; j< r.Max.Y; j++ {
		for i:= r.Min.X ; i < r.Max.X ; i++ {
		
			si := 4*(j * srcwidth + i)
			
			dp[di] = sp[si]
			dp[di+1] = sp[si+1]
			dp[di+2] = sp[si+2]
			dp[di+3] = sp[si+3]
			
			di+=4
		}
		
	}
	
	return dst;
	
}

func makepngs(){
	
	dir, _ := os.Open("p")
	fi, _ := dir.Readdir(200)
	
	for _,f:=range fi {
		fn:= strings.Split(f.Name(), ".")
		//fmt.Println(fn[0])
		
		reader, _ := os.Open("p" + "/" + f.Name())
		f,_ := os.OpenFile("pngs/" + fn[0] + ".png",os.O_CREATE, 0666)
		
		
		convertToPNG(f,reader)
		
		reader.Close()
		f.Close()
		
	}
	
}

func convertImage(m image.Image) (*image.RGBA, error) {
	
	var rgba *image.RGBA
	
	fmt.Println(reflect.TypeOf(m).String())
		
	switch m.(type) {
	case *image.RGBA: 
		rgba=m.(*image.RGBA)
	case *image.YCbCr:
		rgba=YCbCrToRGB(m.(*image.YCbCr))
	default:
		rgba=nil
	}
	
	if rgba!=nil {
		return rgba,nil
	}else{
		return nil,errors.New("tevs")
	}
	
}

func buildDictionary() []MosImage {//map[float32]*image.RGBA {
	
	
	dir, _ := os.Open("jpgs")
	fi, _ := dir.Readdir(200)
	count:=len(fi)
	fmt.Println("count",count)
	
	//dict := make(map[float32]*image.RGBA)
	
	dic := make([]MosImage, 0, 1000)
	

	for i:= 0; i<count;i++ {
		
		reader, err := os.Open("jpgs/" + fi[i].Name())
		if err != nil {
		    log.Fatal(err)
		}
		
		
		m, _, err := image.Decode(reader)
		

		
		if err != nil {
			log.Println(err)
		}
		
		
		rgba,err := convertImage(m)
		if err!=nil {
			log.Println(err)
		}else{
			down:=downsample(rgba,image.Rect(0,0,64,64))
			//lum := averageLum(rgba, rgba.Bounds())
			//dict[lum] = down
			
			mi:=&MosImage{}
			mi.Image=down
			ac:=averageColor(down,down.Bounds())
			mi.AvgColor=&ac
			
			dic=append(dic,*mi)
			
			fmt.Println("lumie :", lum)
		}
		 
		
		reader.Close()

		
	}
	
	return dic
	
}

func lineartosRGB(L float64) uint8 {
	
	//var L float64 = float64(l)/255
	var S float64
	var exp float64 = 1/2.4
	
	
	if L > 0.0031308 {
		S = 1.055*math.Pow(L,exp)-0.055
		
	} else {
		S = L * 12.92
	}
	
	return uint8(255*S)
}

func sRGBtoLinear(s uint8) float64 {
	
	var z float64 = float64(s)/255
		
	var L float64
	
	if z > 0.04045 {
		L = math.Pow((z + 0.055)/(1.055), 2.4)
	} else { 
		L = z/12.92
	}
	
	return L
	
	//uint8(255*L)
	
}
func colorDistance(e1 *color.RGBA, e2 *color.RGBA) float64 {
	
	r:=float64(e1.R)-float64(e2.R)
	g:=float64(e1.G)-float64(e2.G)
	b:=float64(e1.B)-float64(e2.B)
	
	return math.Sqrt(r*r+g*g+b*b)
	
}

func colorDistance2(e1 *color.RGBA, e2 *color.RGBA) float64 {
	//http://www.compuphase.com/cmetric.htm
  rmean := int64(( e1.R + e2.R ) / 2)
  r := int64(e1.R - e2.R);
  g := int64(e1.G - e2.G);
  b := int64(e1.B - e2.B);
	
  return math.Sqrt(float64(  (((512+rmean)*r*r)>>8) + 4*g*g + (((767-rmean)*b*b)>>8)) );
}

