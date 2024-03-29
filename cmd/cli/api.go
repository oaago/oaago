package cli

import (
	"fmt"
	tpl2 "github.com/oaago/oaago/cmd/tpl"
	"github.com/oaago/oaago/const"
	"github.com/oaago/oaago/utils"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
)

func genApi(apiPath, dirName, fileName, method, dec string, met []string) {
	// 验证目录是否存在
	hasDir, _ := utils.PathExists(utils.Camel2Case(apiPath) + utils.Camel2Case(dirName))
	if !hasDir {
		err := os.MkdirAll(utils.GetCurrentPath()+utils.Camel2Case(apiPath)+utils.Camel2Case(dirName), os.ModePerm)
		if err != nil {
			panic("目录初始化失败" + err.Error())
		}
	}
	hasDir1, _ := utils.PathExists(utils.Camel2Case(apiPath) + utils.Camel2Case(dirName) + "/" + utils.Camel2Case(fileName))
	if !hasDir1 {
		err := os.MkdirAll(utils.GetCurrentPath()+utils.Camel2Case(apiPath)+utils.Camel2Case(dirName)+"/"+utils.Camel2Case(fileName), os.ModePerm)
		if err != nil {
			panic("目录初始化失败" + err.Error())
		}
	}
	// 根据types 获取所有文档参数
	types := utils.Camel2Case(_const.ServicePath) + utils.Camel2Case(dirName) + "/" + utils.Camel2Case(fileName)
	fmt.Println("type目录", types)
	_, structList := utils.GetAllStruct(types)
	var param = make(map[string][]string)
	for s, tags := range structList {
		for _, tag := range tags {
			var paramName = tag.Name
			var paramType = tag.Type
			var contentType = "body"
			var required = "false"
			var comment = "-"
			for s2, s3 := range tag.Tags {
				if s2 == "json" {
					paramName = s3
				}
				if s2 == "validate" && strings.Contains(s3, "required") {
					required = "true"
				}
				if s2 == "comment" {
					comment = s3
				}
				if "Get"+utils.Ucfirst(dirName)+utils.Case2Camel(utils.Ucfirst(method))+"Req" == s {
					contentType = "query"
				}
			}
			str := `// @param ` + paramName + " " + contentType + " " + paramType + " " + required + ` "` + comment + `"`
			//fmt.Println("structName: " + s + " param: " + str + "\r\n")
			param[s] = append(param[s], str)
		}
	}
	var Upmet = make([]string, 0)
	for _, s := range met {
		lock := sync.Mutex{}
		for _, funcMap := range _const.SemanticMap {
			lock.Lock()
			if strings.ToLower(funcMap.Method) == strings.ToLower(s) {
				Upmet = append(Upmet, utils.Ucfirst(s))
				HandlerName := strings.Replace(funcMap.FunctionName, "$", utils.Ucfirst(dirName)+utils.Case2Camel(utils.Ucfirst(method)), 1)
				DecMsg := ""
				//增加接口描述根据配置描述接A口
				for k, msg := range _const.DecMessage {
					if s == k {
						DecMsg = strings.Replace(msg, "$", dec, 1)
					}
				}
				// Api 模板变量
				type Api struct {
					Package     string
					UpPackage   string
					Method      string
					UpMethod    string
					Module      string
					HandlerName string
					Param       []string
					Dec         string
					Comment     string
					ServicePath string
					ServiceName string
				}
				apiData := Api{
					Package:     utils.Camel2Case(utils.Lcfirst(dirName) + "_" + utils.Lcfirst(fileName)),
					UpPackage:   utils.Ucfirst(utils.Camel2Case(utils.Lcfirst(dirName) + "_" + utils.Lcfirst(fileName))),
					UpMethod:    utils.Case2Camel(utils.Ucfirst(method)),
					Module:      _const.Module,
					HandlerName: HandlerName,
					Method:      utils.Ucfirst(s),
					Param:       param[HandlerName+"Req"],
					Dec:         DecMsg,
					Comment:     dec,
					ServiceName: utils.Case2Camel(utils.Lcfirst(dirName) + "_" + utils.Lcfirst(fileName)),
					ServicePath: utils.Camel2Case(utils.Lcfirst(dirName) + "/" + utils.Lcfirst(fileName)),
				}
				//创建模板
				fmt.Println("开始api写入模版 " + fileName)
				api := "http-api"
				tmpl := template.New(api)
				//解析模板
				apitext := tpl2.ApiTPL
				tpl, err := tmpl.Parse(apitext)
				if err != nil {
					lock.Unlock()
					panic(err)
				}
				//渲染输出
				filesName := utils.Camel2Case(apiPath) + utils.Camel2Case(dirName) + "/" + utils.Camel2Case(fileName) + "/" + utils.Camel2Case(utils.Lcfirst(HandlerName)) + "_handler.go"
				errs := os.MkdirAll(utils.Camel2Case(apiPath)+utils.Camel2Case(dirName)+"/"+utils.Camel2Case(fileName), os.ModePerm)
				if errs != nil {
					lock.Unlock()
					panic(errs)
				}
				fs, readErr := os.OpenFile(filesName, os.O_RDWR|os.O_CREATE, os.ModePerm)
				if readErr != nil {
					lock.Unlock()
					panic(readErr)
				}
				// 生成文件 渲染模版
				err = tpl.Execute(fs, apiData)
				if err != nil {
					lock.Unlock()
					panic(err)
				}
				ef := fs.Close()
				if ef != nil {
					lock.Unlock()
					panic(ef)
				}
				fmt.Println(dirName + filesName + " api模版创建成功, 开始执行service 创建")
				cmd := exec.Command("gofmt", "-w", filesName)
				ecl := cmd.Run()
				genServerHandler(utils.Camel2Case(utils.Lcfirst(dirName)+"/"+utils.Lcfirst(fileName)), utils.Camel2Case(utils.Lcfirst(dirName)+"_"+utils.Lcfirst(fileName)), utils.Camel2Case(fileName), HandlerName, s)
				//genServerHandlerV2(utils.Camel2Case(utils.Lcfirst(dirName)+"/"+utils.Lcfirst(fileName)), utils.Camel2Case(utils.Lcfirst(dirName)+"_"+utils.Lcfirst(fileName)), utils.Camel2Case(fileName), HandlerName, s)
				if ecl != nil {
					lock.Unlock()
					panic(ecl)
				}

			}
			lock.Unlock()
		}
	}
}

func genServerHandlerV2(dirName, packageName, fileName, funcName string, met string) {
	// 检测目录
	hasDir, _ := utils.PathExists(utils.Camel2Case(_const.ApiServicePath) + utils.Camel2Case(dirName))
	if !hasDir {
		e := os.MkdirAll(_const.ApiServicePath+dirName, os.ModePerm)
		if e != nil {
			panic(e)
		}
	}
	//模板变量
	filesPath := strings.ToLower(utils.Camel2Case(_const.Apifilepath+dirName) + "/" + utils.Camel2Case(utils.Lcfirst(funcName)) + "_handler.go")
	fmt.Println("尝试创建apiv2" + filesPath)
	exists, _ := utils.PathExists(filesPath)
	if exists {
		fmt.Println("service文件已经存在 不会继续创建", filesPath)
		return
	}
	type ApiV2 struct {
		Package   string
		UpPackage string
		Method    string
		UpMethod  string
		Met       string
		Upmet     string
	}
	data := ApiV2{
		Package:   packageName,
		UpPackage: utils.Ucfirst(utils.Case2Camel(packageName)),
		Method:    funcName,
		UpMethod:  utils.Ucfirst(utils.Case2Camel(funcName)),
		Met:       met,
		Upmet:     utils.Ucfirst(met),
	}
	//创建模板
	fmt.Println("开始写入service模版 " + fileName)
	service := "http-service"
	tmpl := template.New(service)
	//解析模板
	apiv2text := tpl2.HttpServiceHandler
	tpl, err := tmpl.Parse(apiv2text)
	if err != nil {
		panic(err)
	}
	hasFile, _ := utils.PathExists(filesPath)
	if hasFile {
		fmt.Println(filesPath + "文件已存在，不会继续创建")
		return
	}
	fs, err1 := os.Create(filesPath)
	if err1 != nil {
		panic(err1)
	}
	err2 := tpl.ExecuteTemplate(fs, service, data)
	if err2 != nil {
		panic(err2)
	}
	fs.Close()
	cmd := exec.Command("gofmt", "-w", filesPath)
	cmd.Run() //nolint:errcheck
	fmt.Println("写入http-service模版成功 " + filesPath)
}
