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
	OP_ORIG = iota + 1
	OP_THUMB
)

type TargetSpec struct {
	Name   string //used as a suffix in output file (also used to identify path in response)
	Op     int    //one of OP_ consts
	Width  int    //note these are ignored by OP_ORIG
	Height int
}

type Coldlink struct {
	StorageDir              string
	MaxOrigImageSizeInBytes int64
}

func (c *Coldlink) Get(remoteURL, localName string, targets []*TargetSpec) (map[string]string, error) {

	results := make(map[string]string)

	tempFilePath, err, deleteFn := c.GetTempImage(remoteURL)
	if err != nil {
		return results, err
	}

	results, err = func() (map[string]string, error) {

		for _, target := range targets {

			switch true {
			case target.Op == OP_THUMB:
				origPath, err := c.MakeThumb(tempFilePath, localName, target.Name, target.Width, target.Height)
				if err != nil {
					return results, err
				}
				results[target.Name] = origPath
			case target.Op == OP_ORIG:
				origPath, err := c.MakeOrig(tempFilePath, localName, target.Name)
				if err != nil {
					return results, err
				}
				results[target.Name] = origPath
			default:
				return results, fmt.Errorf("Unknown target  operation: %s", target.Op)

			}
		}

		//cleanup temp image
		if err := os.Remove(tempFilePath); err != nil {
			return results, err
		}
		return results, nil
	}()

	if deleteErr := deleteFn(); deleteErr != nil {
		if err != nil {
			return nil, fmt.Errorf("Failed: %s (also failed to remove temp image: %s)", err, deleteErr)
		}
	}

	return results, err
}

func (c *Coldlink) GetTempImage(remoteUrl string) (string, error, func() error) {
	response, e := http.Get(remoteUrl)
	if e != nil {
		return "", e, func() error { return nil }
	}
	defer response.Body.Close()

	fileExtension := filepath.Ext(remoteUrl)

	tempfile, err := ioutil.TempFile(os.TempDir(), "cold")
	if err != nil {
		return "", err, func() error { return nil }
	}

	written, err := io.Copy(tempfile, response.Body)
	if err != nil {
		return "", err, func() error { return nil }
	}
	tempfile.Close()

	deleteFn := func() error { return os.Remove(tempfile.Name()) }

	//guard against extremely large image being processed if specified
	if c.MaxOrigImageSizeInBytes > 0 && written > c.MaxOrigImageSizeInBytes {
		toBigErr := fmt.Errorf("Origin image was too big (%d bytes)", written)
		if err := deleteFn(); err != nil {
			toBigErr = fmt.Errorf("%s, also failed to remove temp image because %s", toBigErr.Error(), err.Error())
		}
		return "", toBigErr, func() error { return nil }
	}

	//add extension
	finalName := tempfile.Name() + fileExtension
	if err = os.Rename(tempfile.Name(), finalName); err != nil {
		return "", err, deleteFn
	}

	return finalName, nil, func() error { return os.Remove(finalName) }
}

//MakeOrig just copies the original file somewhere without changing it
func (c *Coldlink) MakeOrig(rawFilePath, localName, suffix string) (string, error) {

	filePath, fileName := c.makeFilePath(localName, suffix, filepath.Ext(rawFilePath))

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

func (c *Coldlink) MakeThumb(rawFilePath, localName, suffix string, width, height int) (string, error) {
	img, err := imaging.Open(rawFilePath)
	if err != nil {
		return "", err
	}
	thumb := imaging.Thumbnail(img, width, height, imaging.CatmullRom)

	filePath, fileName := c.makeFilePath(localName, suffix, filepath.Ext(rawFilePath))
	if err := imaging.Save(thumb, filePath); err != nil {
		return "", err
	}

	return fileName, nil
}

func (c *Coldlink) makeFilePath(localName, typeSuffix, extension string) (string, string) {
	name := fmt.Sprintf("%s.%s%s", localName, typeSuffix, extension)
	return path.Join(c.StorageDir, name), name
}
