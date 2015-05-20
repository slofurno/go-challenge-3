// main_test.go
package main

import (
	"fmt"
	
	"image"
	"os"
	"testing"
)

func openImage(str string) (*image.RGBA,error) {
	
	f,err:= os.Open(str)
	
	if err!=nil{
		return nil,err
	}
	
	defer f.Close()
	
	m,_,_:= image.Decode(f)
	
	img, err:= convertImage(m)
	
	if err!=nil{
		return nil,err
	}
	
	return img,nil
	
}


func TestDownSample(t *testing.T){
	
	m1,err := openImage("test/dl.png")
	
	if err!=nil {
		fmt.Println(err)
		return
	}
	//m2,err := openImage("test/sample.jpg")
	
	
	down := downsample(m1,image.Rect(0,0,m1.Bounds().Max.X/2,m1.Bounds().Max.Y/2))
	
	saveImage(down,"test/downsample.png")
	
	
}
