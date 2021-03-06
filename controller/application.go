package controller

import (
	"archive/zip"
	"fmt"
	"github.com/eaciit/colony-core/v0"
	"github.com/eaciit/colony-manager/helper"
	"github.com/eaciit/knot/knot.v1"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var dest = fmt.Sprintf("%s", filepath.Join(AppBasePath, "..", "colony-app", "apps"))

type ApplicationController struct {
	App
}

func CreateApplicationController(s *knot.Server) *ApplicationController {
	var controller = new(ApplicationController)
	controller.Server = s
	return controller
}

func deleteDirectory(scanDir string, delDir string, dirName string) error {
	dirList, err := ioutil.ReadDir(scanDir)
	if err != nil {
		fmt.Println("Error : ", err)
		return err
	}

	for _, f := range dirList {
		if f.Name() == dirName { /* check if there's already a folder name*/
			err = os.RemoveAll(delDir)
			if err != nil {
				fmt.Println("Error : ", err)
				return err
			}
		}
	}
	return nil
}

func Unzip(src string, newDirName string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		fmt.Println("Error : ", err)
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			fmt.Println("Error : ", err)
			return
		}
	}()

	os.MkdirAll(dest, 0755)
	var basePath string

	extractAndWriteFile := func(i int, f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			fmt.Println("Error : ", err)
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				fmt.Println("Error : ", err)
				return
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				fmt.Println("Error : ", err)
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					fmt.Println("Error : ", err)
					return
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				fmt.Println("Error : ", err)
				return err
			}
		}

		if i == 0 {
			basePath = path
		}
		return nil
	}

	for i, f := range r.File {
		err := extractAndWriteFile(i, f)
		if err != nil {
			fmt.Println("Error : ", err)
			return err
		}
	}

	base := filepath.Base(basePath)
	newname := filepath.Join(dest, strings.Replace(base, base, newDirName, 1))

	err = deleteDirectory(dest, newname, newDirName)
	if err != nil {
		fmt.Println("Error : ", err)
		return err
	}

	err = os.Rename(basePath, newname)
	if err != nil {
		fmt.Println("Error : ", err)
		return err
	}

	return nil
}

func (a *ApplicationController) SaveApps(r *knot.WebContext) interface{} {
	// upload handler
	file, handler, err := r.Request.FormFile("userfile")
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}
	defer file.Close()
	dstSource := fmt.Sprintf("%s", filepath.Join(AppBasePath, "config", "applications", handler.Filename))
	f, err := os.OpenFile(dstSource, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}
	defer f.Close()
	io.Copy(f, file)

	o := new(colonycore.Application)
	o.ID = r.Request.FormValue("id")
	enable, err := strconv.ParseBool(r.Request.FormValue("Enable"))
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}
	o.Enable = enable

	err = colonycore.Delete(o)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	err = colonycore.Save(o)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	if dstSource != "" {
		Unzip(dstSource, o.ID)
	}

	return helper.CreateResult(true, nil, "")
}

func (a *ApplicationController) GetApps(r *knot.WebContext) interface{} {
	r.Config.OutputType = knot.OutputJson

	cursor, err := colonycore.Find(new(colonycore.Application), nil)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	data := []colonycore.Application{}
	err = cursor.Fetch(&data, 0, false)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}
	defer cursor.Close()

	return helper.CreateResult(true, data, "")
}

func (a *ApplicationController) SelectApps(r *knot.WebContext) interface{} {
	r.Config.OutputType = knot.OutputJson

	payload := new(colonycore.Application)
	err := r.GetPayload(payload)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	err = colonycore.Get(payload, payload.ID)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	return helper.CreateResult(true, payload, "")
}

func (a *ApplicationController) DeleteApps(r *knot.WebContext) interface{} {
	r.Config.OutputType = knot.OutputJson

	payload := new(colonycore.Application)
	err := r.GetPayload(payload)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	err = colonycore.Delete(payload)
	if err != nil {
		return helper.CreateResult(false, nil, err.Error())
	}

	delPath := filepath.Join(dest, payload.ID)
	err = deleteDirectory(dest, delPath, payload.ID)
	if err != nil {
		fmt.Println("Error : ", err)
		return err
	}

	return helper.CreateResult(true, nil, "")
}
