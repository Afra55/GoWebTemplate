package main

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

const (
	ListDir    = 0x0001
	UPLOAD_DIR = "./src/uploads"
	VIEWS_DIR  = "./src/views"
	STATIC_DIR = "./src/public"
)

// 全局变量 templates ， 用于存放所有模板内容
var templates = make(map[string]*template.Template)

type Man struct {
	Age  int
	Name string
	Sex  bool
}

// 初始化，会在 main 函数之前运行, 遍历解析 views 目录下的所有html，存入 templates 切片中
func init() {
	fileInfoArr, err := ioutil.ReadDir(VIEWS_DIR)
	if err != nil {
		panic(err)
		return
	}
	var fileName, filePath string
	for _, fileInfo := range fileInfoArr {
		fileName = fileInfo.Name()
		if ext := path.Ext(fileName); ext != ".html" {
			continue
		}
		filePath = VIEWS_DIR + "/" + fileName
		log.Println("loading template : " + filePath)
		t := template.Must(template.ParseFiles(filePath))
		templates[fileName] = t
	}

}

func main() {
	// mux 可以同时加载静态和 动态网页
	mux := http.NewServeMux()

	// 静态网页 localhost:8080/assets/
	staticDirHandler(mux, "/assets/", STATIC_DIR, 0)

	// localhost:8080/upload
	mux.HandleFunc("/upload", safeHandler(uploadHandler))

	// localhost:8080/view
	mux.HandleFunc("/view", safeHandler(viewImageHandler))

	// localhost:8080/list
	mux.HandleFunc("/list", safeHandler(listSavedImage))

	// localhost:8080/json
	mux.HandleFunc("/json", safeHandler(textJsonReturn))
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err.Error())
	}
}

// 用于加载静态变量
func staticDirHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix)-1:]
		if (flags & ListDir) == 0 {
			if exists := IsExist(file); !exists {
				http.NotFound(w, r)
				return
			}
		}
		http.ServeFile(w, r, file)
	})
}

func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

// 避免方法出错导致程序停止运行
func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// 业务逻辑处理函数里边引发了panic，就会调用 recover() 对其进行检测, 避免 panic 造成程序停止
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
			}
		}()
		fn(w, r)
	}
}

// 公共方法，解析模板文件
func useTemplateFile(w http.ResponseWriter, fileName string, locals map[string]interface{}) (err error) {
	resultErr := templates[fileName+".html"].Execute(w, locals)
	return resultErr
}

func textJsonReturn(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		me := Man{
			12,
			"Afra",
			true,
		}
		jsonStr, err := json.Marshal(me)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonStr)
	}
}

/* 查看单张图片 */
func viewImageHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, imagePath)
}

/* 列出所有保存的图片 */
func listSavedImage(w http.ResponseWriter, r *http.Request) {
	fileInfoArr, err := ioutil.ReadDir(UPLOAD_DIR)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	/* 下面的方法是不适用模板的方法 */
	//var listHtml string
	//for _, file := range fileInfoArr{
	//	imageId := file.Name()
	//	listHtml += "<li><a href=\"/view?id="+ imageId+"\">" + imageId +"</a></li>"
	//}
	//io.WriteString(w, "<p><ol>"+listHtml+"</ol></p>")

	/* 下面的方法是用 list 模板*/
	locals := make(map[string]interface{})
	images := []string{}
	for _, file := range fileInfoArr {
		images = append(images, file.Name())
	}
	locals["images"] = images
	resultErr := useTemplateFile(w, "list", locals)
	if resultErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// 图片上传方法
func uploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		// 用于展示上传页面

		err := useTemplateFile(w, "upload", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return

	} else if r.Method == "POST" {
		// 用于上传图片

		// 获取文件的信息
		f, h, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileName := h.Filename

		defer f.Close()

		// 在 views 目录下创建 占位文件
		tempFile, err := os.Create(UPLOAD_DIR + "/" + fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer tempFile.Close()

		// 把上传的文件复制并覆盖 占位文件
		if _, err := io.Copy(tempFile, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 重定向去查看上传的单张图片
		http.Redirect(w, r, "/view?id="+fileName, http.StatusFound)
	}
}
