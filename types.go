// types

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type FieldInfo struct {
	Index    int          // 对应的excel列序号
	Desc     string       // 字段描述
	Name     string       // 字段名
	Type     string       // 字段数据类型
	RawType  string       // 原始字段数据类型
	Mode     string       // 生成方式(s=server,c=client,b=both)
	Deepth   int          // 字段深度
	Array    bool         // 是否是数组字段
	ArrayIdx int          // 数组索引
	Parent   *FieldInfo   // 父字段
	Root     *FieldInfo   // 根字段
	Fields   []*FieldInfo // 成员
	Row      []string     // 解析是
}

type Xlsx struct {
	Name      string
	Comments  []string
	Data      []string
	RootField *FieldInfo
	Descs     []string
	Names     []string
	Types     []string
	Modes     []string
	Rows      [][]string
}

// 解析头部
func (x *Xlsx) parseHeader() {
	// 是否有id
	keyName := x.Names[0]
	if keyName != "id" {
		fmt.Println(keyName)
		panic("key 必须以 id 命名")
	}
	for i, def := range x.Types {
		if len(strings.TrimSpace(def)) == 0 {
			panic(fmt.Sprintf("列[%d]数据类型为空", i))
		}
	}

	rootField := new(FieldInfo)
	rootField.Index = -1
	rootField.Fields = make([]*FieldInfo, 0, len(x.Names))
	x.RootField = rootField

	for idx := 0; idx < len(x.Types); idx++ {
		idx = x.parseField(rootField, idx, -1)
	}
	fmt.Println(rootField, math.MaxInt)
}

func (x *Xlsx) parseField(parent *FieldInfo, index, arrayIdx int) int {
	if index >= len(x.Types) {
		return index
	}

	def := strings.TrimSpace(x.Types[index])
	field := new(FieldInfo)
	field.Index = index
	field.RawType = def
	field.Parent = parent
	field.Deepth = parent.Deepth + 1
	field.ArrayIdx = arrayIdx
	field.Root = parent.Root
	if field.Root == nil && parent.Index == -1 {
		field.Root = parent
	}
	if len(x.Descs) > index {
		field.Desc = x.Descs[index]
	}
	if len(x.Names) > index {
		field.Name = x.Names[index]
	}
	if len(x.Modes) > index {
		field.Mode = x.Modes[index]
	}
	parent.Fields = append(parent.Fields, field)

	if arrayBegin := strings.Index(def, "["); arrayBegin != -1 {
		// array
		field.Type = def[:arrayBegin]
		field.Array = true

		// sub array
		arrayEnd := strings.Index(def, "]")
		fieldNum, _ := strconv.Atoi(def[(arrayBegin + 1):arrayEnd])
		for i := 0; i < fieldNum; i++ {
			index = x.parseField(field, index+1, i)
		}
	} else {
		field.Type = def

		isDict := strings.HasPrefix(def, "dict")
		if isDict {
			dictBegin := strings.Index(def, "<")
			dictEnd := strings.Index(def, ">")
			fieldNum, _ := strconv.Atoi(def[(dictBegin + 1):dictEnd])
			for i := 0; i < fieldNum; i++ {
				index = x.parseField(field, index+1, -1)
			}
		}
		if parent.Array {
			if field.Type != parent.Type {
				//panic(errors.New("类型不匹配"))
			}
		}
	}
	return index
}

// id 必须是int
// id 冲突
// 字段名冲突（同层）
// 类型检查(例如: int 类型的字段填了 string， 耗性能)
// 高级特性：id公式，数值范围检查，字段注释，配置行注释
func (x *Xlsx) parseRows(rows [][]string) {
	// comment
	x.Comments = make([]string, 0)
	for _, field := range x.RootField.Fields {
		x.parseComment(field)
	}
	fmt.Println(strings.Join(x.Comments, "\n"))

	// data
	x.Data = make([]string, 0)
	for line := 4; line < len(rows); line++ {
		row := rows[line]
		if strings.HasPrefix(row[0], "//") || row[0] == "" {
			continue
		}
		x.parseRow(x.RootField, row, -1, 0)
	}
	fmt.Println(strings.Join(x.Data, "\n"))
}

func (x *Xlsx) parseComment(f *FieldInfo) {
	var comment string
	if f.Parent.Array {
		comment = fmt.Sprintf("-- %-30s %-10s %s", strings.Repeat(" ", f.Deepth*2)+"["+strconv.Itoa(f.ArrayIdx)+"]", f.RawType, f.Desc)
	} else {
		comment = fmt.Sprintf("-- %-30s %-10s %s", strings.Repeat(" ", f.Deepth*2)+f.Name, f.RawType, f.Desc)
	}
	x.Comments = append(x.Comments, comment)
	if len(f.Fields) > 0 {
		for _, field := range f.Fields {
			x.parseComment(field)
		}
	}
}

func (x *Xlsx) parseRow(f *FieldInfo, row []string, index, deepth int) {
	if f.Index == -1 {
		// root
		x.Data = append(x.Data, fmt.Sprintf("[%s] = {", row[0]))
	} else {
		if len(f.Fields) > 0 {
			if f.Parent.Array {
				x.Data = append(x.Data, fmt.Sprintf("%s[%d] = {", strings.Repeat(" ", deepth*2), index))
			} else {
				x.Data = append(x.Data, fmt.Sprintf("%s%s = {", strings.Repeat(" ", deepth*2), f.Name))
			}
		} else {
			if f.Parent.Array {
				x.Data = append(x.Data, fmt.Sprintf("%s[%d] = %s,", strings.Repeat(" ", deepth*2), index, row[f.Index]))
			} else {
				x.Data = append(x.Data, fmt.Sprintf("%s%s = %s,", strings.Repeat(" ", deepth*2), f.Name, row[f.Index]))
			}
		}
	}
	for idx, field := range f.Fields {
		x.parseRow(field, row, idx+1, deepth+1)
	}
	if len(f.Fields) > 0 {
		x.Data = append(x.Data, fmt.Sprintf("%s},", strings.Repeat(" ", deepth*2)))
	}
}
