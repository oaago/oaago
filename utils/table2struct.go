package utils

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_const2 "github.com/oaago/oaago/const"
	"log"
	"os"
	"os/exec"
	"strings"
)

//map for converting mysql type to golang types
var typeForMysqlToGo = map[string]string{
	"int":                "int64",
	"integer":            "int64",
	"tinyint":            "int64",
	"smallint":           "int64",
	"mediumint":          "int64",
	"bigint":             "int64",
	"int unsigned":       "int64",
	"integer unsigned":   "int64",
	"tinyint unsigned":   "int64",
	"smallint unsigned":  "int64",
	"mediumint unsigned": "int64",
	"bigint unsigned":    "int64",
	"bit":                "int64",
	"bool":               "bool",
	"enum":               "string",
	"set":                "string",
	"varchar":            "string",
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string",
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"date":               "time.Time", // time.Time or string
	"datetime":           "time.Time", // time.Time or string
	"timestamp":          "time.Time", // time.Time or string
	"time":               "time.Time", // time.Time or string
	"float":              "float64",
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string",
	"varbinary":          "string",
}

type Table2Struct struct {
	dsn            string
	savePath       string
	db             *sql.DB
	table          string
	prefix         string
	config         *T2tConfig
	err            error
	realNameMethod string
	enableJsonTag  bool   // 是否添加json的tag, 默认不添加
	packageName    string // 生成struct的包名(默认为空的话, 则取名为: package model)
	tagKey         string // tag字段的key值,默认是orm
	dateToTime     bool   // 是否将 date相关字段转换为 time.Time,默认否
}

type T2tConfig struct {
	StructNameToHump bool // 结构体名称是否转为驼峰式，默认为false
	RmTagIfUcFirsted bool // 如果字段首字母本来就是大写, 就不添加tag, 默认false添加, true不添加
	TagToLower       bool // tag的字段名字是否转换为小写, 如果本身有大写字母的话, 默认false不转
	JsonTagToHump    bool // json tag是否转为驼峰，默认为false，不转换
	UcFirstOnly      bool // 字段首字母大写的同时, 是否要把其他字母转换为小写,默认false不转换
	SeperatFile      bool // 每个struct放入单独的文件,默认false,放入同一个文件
}

func NewTable2Struct() *Table2Struct {
	return &Table2Struct{}
}

func (t *Table2Struct) Dsn(d string) *Table2Struct {
	t.dsn = d
	return t
}

func (t *Table2Struct) TagKey(r string) *Table2Struct {
	t.tagKey = r
	return t
}

func (t *Table2Struct) PackageName(r string) *Table2Struct {
	t.packageName = r
	return t
}

func (t *Table2Struct) RealNameMethod(r string) *Table2Struct {
	t.realNameMethod = r
	return t
}

func (t *Table2Struct) SavePath(p string) *Table2Struct {
	t.savePath = p
	return t
}

func (t *Table2Struct) DB(d *sql.DB) *Table2Struct {
	t.db = d
	return t
}

func (t *Table2Struct) Table(tab string) *Table2Struct {
	t.table = tab
	return t
}

func (t *Table2Struct) Prefix(p string) *Table2Struct {
	t.prefix = p
	return t
}

func (t *Table2Struct) EnableJsonTag(p bool) *Table2Struct {
	t.enableJsonTag = p
	return t
}

func (t *Table2Struct) DateToTime(d bool) *Table2Struct {
	t.dateToTime = d
	return t
}

func (t *Table2Struct) Config(c *T2tConfig) *Table2Struct {
	t.config = c
	return t
}

func (t *Table2Struct) Run() (map[string]map[string]string, error) {
	var tableInfo = make(map[string]map[string]string)
	if t.config == nil {
		t.config = new(T2tConfig)
	}
	// 链接mysql, 获取db对象
	t.dialMysql()
	if t.err != nil {
		return nil, t.err
	}

	// 获取表和字段的shcema
	tableColumns, err := t.getColumns()
	if err != nil {
		return nil, err
	}

	// 包名
	var packageName string
	if t.packageName == "" {
		packageName = "package model\n\n"
	} else {
		packageName = fmt.Sprintf("package %s\n\n", t.packageName)
	}

	// 组装struct
	var structContent, getStructContent, MapStructStr string
	for tableRealName, item := range tableColumns {
		// 去除前缀
		if t.prefix != "" {
			tableRealName = tableRealName[len(t.prefix):]
		}
		tableName := tableRealName
		structName := tableName
		if t.config.StructNameToHump {
			structName = t.camelCase(structName)
		}

		switch len(tableName) {
		case 0:
		case 1:
			tableName = strings.ToUpper(tableName[0:1])
		default:
			// 字符长度大于1时
			tableName = strings.ToUpper(tableName[0:1]) + tableName[1:]
		}
		depth := 1

		// 默认 struct 内容
		structContent += "type " + structName + " struct {\n"
		for _, v := range item {
			//structContent += tab(depth) + v.ColumnName + " " + v.Type + " " + v.Tag + "\n"
			if v.ColumnName == "CreateTime" || v.ColumnName == "UpdateTime" {
				continue
			}
			// 字段注释
			var clumnComment string
			if v.ColumnComment != "" {
				clumnComment = fmt.Sprintf(" // %s", v.ColumnComment)
			} else {
				clumnComment = "//-"
			}
			if len(clumnComment) > 0 {
				clumnComment = strings.Replace(clumnComment, "\n", "", -1)
				clumnComment = strings.Replace(clumnComment, "\r", "", -1)
				clumnComment = strings.Replace(clumnComment, " ", "", -1)
			}
			if v.ColumnName != "CreateTime" && v.ColumnName != "UpdateTime" {
				tag := "`" + strings.Replace(v.Tag, "`", "", -1) + " validate:\"required\"" + " comment:\"" + strings.Replace(clumnComment, "//", "", 1) + "\"`"
				structContent += fmt.Sprintf("%s%s %s %s%s\n",
					tab(depth), v.ColumnName, v.Type, tag, clumnComment)
			}
			info := make(map[string]string)
			info["type"] = v.Type
			info["comment"] = clumnComment
			tableInfo[v.ColumnName] = info
		}
		structContent += tab(depth-1) + "}\n\n"
		// get和delete的请求
		getStructContent += "type " + structName + " struct {\n"
		getStructContent += "   Id int64  `json:\"" + "id\" validate:\"required\" form:\"" + "id\"`\n"
		getStructContent += tab(depth-1) + "}\n\n"
		for _, funcMap := range _const2.SemanticMap {
			ReqName := strings.Replace(funcMap.FunctionName, "$", Case2Camel(Ucfirst(structName)), 1)
			ResName := strings.Replace(funcMap.FunctionName, "$", Case2Camel(Ucfirst(structName)), 1)
			MapStructStr = MapStructStr + strings.Replace(structContent, structName, ReqName+"Req", 1) + strings.Replace(structContent, structName, ResName+"Res", 1)
		}

		//patchStructContent = structContent
		//putStructContent = patchStructContent
		//postStructContent = patchStructContent
		//deleteStructContent = getStructContent
		//
		//// res
		//getStructContentRes = postStructContent
		//putStructContentRes = getStructContentRes
		//patchStructContentRes = getStructContentRes
		//postStructContentRes = getStructContentRes
		//deleteStructContentRes = getStructContentRes
		//
		//patchStructContent = strings.Replace(patchStructContent, structName, "Patch"+Case2Camel(Ucfirst(structName))+"Req", 1)
		//putStructContent = strings.Replace(putStructContent, structName, "Put"+Case2Camel(Ucfirst(structName))+"Req", 1)
		//getStructContent = strings.Replace(getStructContent, structName, "Get"+Case2Camel(Ucfirst(structName))+"Req", 1)
		//postStructContent = strings.Replace(postStructContent, structName, "Post"+Case2Camel(Ucfirst(structName))+"Req", 1)
		//deleteStructContent = strings.Replace(deleteStructContent, structName, "Delete"+Case2Camel(Ucfirst(structName))+"Req", 1)
		//
		//getStructContentRes = strings.Replace(getStructContentRes, structName, "Get"+Case2Camel(Ucfirst(structName))+"Res", 1)
		//putStructContentRes = strings.Replace(putStructContentRes, structName, "Put"+Case2Camel(Ucfirst(structName))+"Res", 1)
		//postStructContentRes = strings.Replace(postStructContentRes, structName, "Post"+Case2Camel(Ucfirst(structName))+"Res", 1)
		//patchStructContentRes = strings.Replace(patchStructContentRes, structName, "Patch"+Case2Camel(Ucfirst(structName))+"Res", 1)
		//deleteStructContentRes = strings.Replace(deleteStructContentRes, structName, "Delete"+Case2Camel(Ucfirst(structName))+"Res", 1)
		// 添加 method 获取真实表名
		if t.realNameMethod != "" {
			structContent += fmt.Sprintf("func (%s) %s() string {\n",
				structName, t.realNameMethod)
			structContent += fmt.Sprintf("%sreturn \"%s\"\n",
				tab(depth), tableRealName)
			structContent += "}\n\n"
		}
	}

	// 如果有引入 time.Time, 则需要引入 time 包
	var importContent string
	if strings.Contains(structContent, "time.Time") {
		importContent = "import \"time\"\n\n"
	}

	// 写入文件struct
	var savePath = t.savePath
	// 是否指定保存路径
	if savePath == "" {
		savePath = "model.go"
	}
	filePath := fmt.Sprintf("%s", savePath)
	f, err := os.Create(filePath)
	if err != nil {
		log.Println("Can not write file")
		return nil, err
	}
	defer f.Close()
	f.WriteString(packageName + importContent + MapStructStr) //nolint:errcheck
	cmd := exec.Command("gofmt", "-w", filePath)
	e := cmd.Run()
	if e != nil {
		return nil, e
	} //nolint:errcheck
	log.Println("gen model finish!!!")
	return tableInfo, nil
}

func (t *Table2Struct) dialMysql() {
	if t.db == nil {
		if t.dsn == "" {
			t.err = errors.New("dsn数据库配置缺失")
			return
		}
		t.db, t.err = sql.Open("mysql", t.dsn)
	}
	return
}

type column struct {
	ColumnName    string
	Type          string
	Nullable      string
	TableName     string
	ColumnComment string
	Tag           string
}

// Function for fetching schema definition of passed table
func (t *Table2Struct) getColumns() (tableColumns map[string][]column, err error) {
	// 根据设置,判断是否要把 date 相关字段替换为 string
	if t.dateToTime == false {
		typeForMysqlToGo["date"] = "string"
		typeForMysqlToGo["datetime"] = "string"
		typeForMysqlToGo["timestamp"] = "string"
		typeForMysqlToGo["time"] = "string"
	}
	tableColumns = make(map[string][]column)
	// sql
	var sqlStr = `SELECT COLUMN_NAME,DATA_TYPE,IS_NULLABLE,TABLE_NAME,COLUMN_COMMENT
		FROM information_schema.COLUMNS 
		WHERE table_schema = DATABASE()`
	// 是否指定了具体的table
	if t.table != "" {
		sqlStr += fmt.Sprintf(" AND TABLE_NAME = '%s'", t.prefix+t.table)
	}
	// sql排序
	sqlStr += " order by TABLE_NAME asc, ORDINAL_POSITION asc"
	rows, err := t.db.Query(sqlStr)
	if err != nil {
		log.Println("Error reading table information: ", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		col := column{}
		err = rows.Scan(&col.ColumnName, &col.Type, &col.Nullable, &col.TableName, &col.ColumnComment)

		if err != nil {
			log.Println(err.Error())
			return
		}
		col.Tag = col.ColumnName
		col.ColumnName = t.camelCase(col.ColumnName)
		col.Type = typeForMysqlToGo[col.Type]
		jsonTag := col.Tag
		// 字段首字母本身大写, 是否需要删除tag
		if t.config.RmTagIfUcFirsted &&
			col.ColumnName[0:1] == strings.ToUpper(col.ColumnName[0:1]) {
			col.Tag = "-"
		} else {
			// 是否需要将tag转换成小写
			if t.config.TagToLower {
				col.Tag = strings.ToLower(col.Tag)
				jsonTag = col.Tag
			}

			if t.config.JsonTagToHump {
				jsonTag = t.camelCase(jsonTag)
			}

			//if col.Nullable == "YES" {
			//	col.Json = fmt.Sprintf("`json:\"%s,omitempty\"`", col.Json)
			//} else {
			//}
		}
		if t.tagKey == "" {
			t.tagKey = "orm"
		}
		if t.enableJsonTag {
			//col.Json = fmt.Sprintf("`json:\"%s\" %s:\"%s\"`", col.Json, t.config.TagKey, col.Json)
			//col.Tag = fmt.Sprintf("`%s:\"%s\" json:\"%s\"`", t.tagKey, col.Tag, jsonTag)
			col.Tag = fmt.Sprintf("`json:\"%s\"`", Lcfirst(Case2Camel(jsonTag)))
		} else {
			col.Tag = fmt.Sprintf("`%s:\"%s\"`", t.tagKey, col.Tag)
		}
		//columns = append(columns, col)
		if _, ok := tableColumns[col.TableName]; !ok {
			tableColumns[col.TableName] = []column{}
		}
		tableColumns[col.TableName] = append(tableColumns[col.TableName], col)
	}
	return
}

func (t *Table2Struct) camelCase(str string) string {
	// 是否有表前缀, 设置了就先去除表前缀
	if t.prefix != "" {
		str = strings.Replace(str, t.prefix, "", 1)
	}
	var text string
	//for _, p := range strings.Split(name, "_") {
	for _, p := range strings.Split(str, "_") {
		// 字段首字母大写的同时, 是否要把其他字母转换为小写
		switch len(p) {
		case 0:
		case 1:
			text += strings.ToUpper(p[0:1])
		default:
			// 字符长度大于1时
			if t.config.UcFirstOnly == true {
				text += strings.ToUpper(p[0:1]) + strings.ToLower(p[1:])
			} else {
				text += strings.ToUpper(p[0:1]) + p[1:]
			}
		}
	}
	return text
}
func tab(depth int) string {
	return strings.Repeat("\t", depth)
}
