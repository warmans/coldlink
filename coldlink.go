package coldlink

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/disintegration/imaging"
)

const (
	OPT_ORIG = "orig" //original dimensions
	OPT_SM   = "sm"   //150x150
	OPT_XS   = "xs"   //50x50
)

type MakeFn func(rawFilePath, localName string) (string, error)

type Coldlink struct {
	StorageDir string
}

func (c *Coldlink) Get(remoteURL, localName string, targets []string) (map[string]string, error) {

	results := make(map[string]string)

	tempFilePath, err := c.GetTempImage(remoteURL)
	if err != nil {
		return results, err
	}

	optFuncMap := map[string]MakeFn{
		OPT_ORIG: c.MakeOrig,
		OPT_SM:   c.MakeSm,
		OPT_XS:   c.MakeXs,
	}

	for _, target := range targets {
		makeFn, ok := optFuncMap[target]
		if ok == false {
			return results, fmt.Errorf("Unknown target %s", target)
		}
		origPath, err := makeFn(tempFilePath, localName)
		if err != nil {
			return results, err
		}
		results[target] = origPath
	}

	return results, nil
}

func (c *Coldlink) GetTempImage(remoteUrl string) (string, error) {
	response, e := http.Get(remoteUrl)
	if e != nil {
		return "", e
	}
	defer response.Body.Close()

	fileExtension := filepath.Ext(remoteUrl)

	tempfile, err := ioutil.TempFile(os.TempDir(), "cold")
	if err != nil {
		return "", err
	}

	_, err = io.Copy(tempfile, response.Body)
	if err != nil {
		return "", err
	}
	tempfile.Close()

	//add extension
	finalName := tempfile.Name() + fileExtension
	if err = os.Rename(tempfile.Name(), finalName); err != nil {
		return "", err
	}

	return finalName, nil
}

//MakeOrig just copies the original file somewhere without changing it
func (c *Coldlink) MakeOrig(rawFilePath, localName string) (string, error) {

	filePath, fileName := c.makeFilePath(localName, OPT_ORIG, filepath.Ext(rawFilePath))

	srcFile, err := os.Open(rawFilePath)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	destFile, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func (c *Coldlink) MakeSm(rawFilePath, localName string) (string, error) {
	return c.MakeThumb(rawFilePath, localName, OPT_SM, 150, 150)
}

func (c *Coldlink) MakeXs(rawFilePath, localName string) (string, error) {
	return c.MakeThumb(rawFilePath, localName, OPT_XS, 50, 50)
}

func (c *Coldlink) MakeThumb(rawFilePath, localName, typeName string, width, height int) (string, error) {
	img, err := imaging.Open(rawFilePath)
	if err != nil {
		return "", err
	}
	thumb := imaging.Thumbnail(img, width, height, imaging.CatmullRom)

	filePath, fileName := c.makeFilePath(localName, typeName, filepath.Ext(rawFilePath))
	if err := imaging.Save(thumb, filePath); err != nil {
		return "", err
	}

	return fileName, nil
}

func (c *Coldlink) makeFilePath(localName, typeSuffix, extension string) (string, string) {
	name := fmt.Sprintf("%s.%s%s", localName, typeSuffix, extension)
	return path.Join(c.StorageDir, name), name
}
