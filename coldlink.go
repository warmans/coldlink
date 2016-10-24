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

// Options for image targets
const (
	OpOriginal = iota + 1
	OpThumb
)

// TargetSpec describes an image output target
type TargetSpec struct {
	Name   string //used as a suffix in output file (also used to identify path in response)
	Op     int    //one of OP_ consts
	Width  int    //note these are ignored by OP_ORIG
	Height int
}

// Coldlink handles fetching, resizing, and storing images from remote URLs
type Coldlink struct {
	StorageDir              string
	MaxOrigImageSizeInBytes int64
}

// TempFile is an image file that is deleted when closed
// this is handy for ReadCloser implementations of temporary files
type TempFile struct {
	*os.File
}

// Close closes the file handle then deletes the underlying file
func (t *TempFile) Close() error {
	err := t.File.Close()
	if err != nil {
		return err
	}
	return os.Remove(t.Name())
}

// Get an image, storing it locally in the defned target formats
func (c *Coldlink) Get(remoteURL, localName string, targets []*TargetSpec) (map[string]string, error) {

	results := make(map[string]string)

	tempFile, err := c.GetTempImage(remoteURL)
	if err != nil {
		return results, err
	}

	results, err = func() (map[string]string, error) {

		for _, target := range targets {

			switch true {
			case target.Op == OpThumb:
				origPath, err := c.MakeThumb(tempFile.Name(), localName, target.Name, target.Width, target.Height)
				if err != nil {
					return results, err
				}
				results[target.Name] = origPath
			case target.Op == OpOriginal:
				origPath, err := c.MakeOrig(tempFile.Name(), localName, target.Name)
				if err != nil {
					return results, err
				}
				results[target.Name] = origPath
			default:
				return results, fmt.Errorf("Unknown target  operation: %v", target.Op)
			}
		}
		return results, nil
	}()
	if err != nil {
		closeErr := tempFile.Close()
		if closeErr != nil {
			return results, fmt.Errorf("two errors: processing request: %s, closing temp file: %s", err, closeErr)
		}
		return results, err
	}

	//cleanup temp image
	if err := tempFile.Close(); err != nil {
		return results, err
	}

	return results, err
}

// GetTempImage retrieves an image and stores it in a temporary file
func (c *Coldlink) GetTempImage(remoteURL string) (*TempFile, error) {
	response, err := http.Get(remoteURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	tf, err := ioutil.TempFile(os.TempDir(), "cold")
	if err != nil {
		return nil, err
	}
	tempFile := &TempFile{tf}

	written, err := io.Copy(tempFile, response.Body)
	if err != nil {
		return nil, err
	}

	//guard against extremely large image being processed if specified
	if c.MaxOrigImageSizeInBytes > 0 && written > c.MaxOrigImageSizeInBytes {
		tooBigErr := fmt.Errorf("Origin image was too big (%d bytes)", written)
		if err := tempFile.Close(); err != nil {
			tooBigErr = fmt.Errorf("%s, also failed to remove temp image because %s", tooBigErr, err)
		}
		return nil, tooBigErr
	}
	//add extension
	finalName := tempFile.Name() + filepath.Ext(remoteURL)
	if err = os.Rename(tempFile.Name(), finalName); err != nil {
		closeErr := tempFile.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("error renaming and closing temp file: %s - %s", err, closeErr)
		}
		return nil, err
	}
	tempFile.File.Close()
	tempFile.File, err = os.Open(finalName)
	return tempFile, err
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

// MakeThumb creates a thumbnail image with the given width and height
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
