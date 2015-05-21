// main_test.go
package main

import (
	"fmt"
	"strconv"
	"image/color"
	"image/draw"
	"time"
	"math/rand"
	
	
	"image"
	"os"
	"testing"
)

func init(){
	 rand.Seed( time.Now().UTC().UnixNano())
}

func openImage(str string) (*image.RGBA,error) {
	
	f,err:= os.Open(str)
	
	if err!=nil{
		return nil,err
	}
	
	defer f.Close()
	
	m,_,err:= image.Decode(f)
	
	if err!=nil{
		
		return nil,err
	}
	
	img:= convertImage(m)
		
	return img,nil
	
}


func TestConvert(t *testing.T) {
	
	const DIR = "testsrc"
	
	dir,_:=os.Open(DIR)
	fi,_ :=dir.Readdir(10)
	
	for _,f:=range fi {
		
		_,err:=openImage(DIR + "/" + f.Name())
		
		if err!=nil {
			t.Error(err)
		}
		
	}
	
}

func TestFit (t *testing.T){
	
	org,_:=openImage("test/testfit.png")
	var tiles []MosImage
	const DIR = "testtiles"
	
	dir,_:=os.Open(DIR)
	fi,_ :=dir.Readdir(100)
	
	for _,f:=range fi {
		
		m,err:=openImage(DIR + "/" + f.Name())
		
		if err!=nil {
			t.Error(err)
		}
		
		mr:=NewMosImage(m)
		tiles = append(tiles,mr)
		
		
		
	}
	
	src,_:=openImage("test/bm.jpg")
	
	mr:=fitMosaic(src,tiles)	
	tevs,_,_ := image.Decode(mr.Mosaic)
	
	if !isImageEqual(tevs,org){
		t.Error("not eql")
	}
	
	out:=convertImage(tevs)	
	saveImage(out, "testtest.png")
	
}

func isImageEqual(m1 image.Image, m2 image.Image) bool{
	
	if m1.Bounds().Max.X != m2.Bounds().Max.X || m1.Bounds().Max.Y != m2.Bounds().Max.Y {
		return false
	}
	
	dif:=0
	
	for i:=0;i<m1.Bounds().Max.X;i++ {
		for j:=0;j<m1.Bounds().Max.Y;j++ {
			
			r1,g1,b1,_:= m1.At(i,j).RGBA()
			r2,g2,b2,_:= m2.At(i,j).RGBA()
			
			if r1!=r2 || g1!=g2 || b1!=b2 {
				dif++
			}
			
		}
	}
	
	fmt.Println("count: ",dif," ?? ", m1.Bounds().Max.X*m1.Bounds().Max.Y)
	
	return true
}

func generateTiles (){
	
	
	for i:= 0; i < 100; i++ {
		
		x:=rand.Intn(400)+64
		y:=rand.Intn(400)+64
		
		r:=uint8(rand.Intn(256))
		g:=uint8(rand.Intn(256))
		b:=uint8(rand.Intn(256))
		
		c := color.RGBA{r, g, b, 255}
		
		tile:= image.NewRGBA(image.Rect(0,0,x,y))
		
		draw.Draw(tile, tile.Bounds(), &image.Uniform{c}, image.ZP, draw.Src)
		
		saveImage(tile,"testtiles/"+strconv.Itoa(i)+".png")
		
		
	}
	
	
	
}

func TestDownSample(t *testing.T){
	
	m1,err := openImage("test/dl.png")
	
	if err!=nil {
		t.Error(err)
		return
	}
	//m2,err := openImage("test/sample.jpg")
	
	
	down := downsample(m1,image.Rect(0,0,m1.Bounds().Max.X/2,m1.Bounds().Max.Y/2))
	
	saveImage(down,"test/downsample.png")
	
	
}
