package stringify

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cast"
)

const (
	defaultTagName = "json"
)

var (
	bufferSize = 512
)

var (
	jsPool = &sync.Pool{
		New: func() interface{} {
			return &JSONStringify{
				Sb: bytes.NewBuffer(make([]byte, 0, bufferSize)),
			}
		},
	}
)

type (
	// JSONStringify json stringify
	JSONStringify struct {
		Sb       *bytes.Buffer
		TagName  string
		Replacer Replacer
	}
	// Replacer replace function
	Replacer func(key string, value interface{}) (replace bool, newValue string)
)

// GetBufferSize get initialize buffer size
func GetBufferSize() int {
	return bufferSize
}

// SetBufferSize set initialize buffer size
func SetBufferSize(size int) {
	bufferSize = size
}

func isIgnore(v reflect.Value) bool {
	return v.Kind() == reflect.Invalid
}

// St stringify struct
func (js *JSONStringify) St(s interface{}, embedded bool) {
	sb := js.Sb
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	len := t.NumField()
	if !embedded {
		sb.WriteRune('{')
	}
	first := true
	for i := 0; i < len; i++ {

		field := t.Field(i)
		// we can't access the value of unexported fields
		if field.PkgPath != "" {
			continue
		}

		key := field.Name
		value := v.FieldByIndex(field.Index)

		// 如果是指针，获取真实值
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		// 如果是需要忽略的，则忽略
		if isIgnore(value) {
			continue
		}

		// 从tag的配置中获取名字
		var tagName string
		if js.TagName != "" {
			tagName = js.TagName
		} else {
			tagName = defaultTagName
		}
		tag := field.Tag.Get(tagName)
		// 如果忽略则跳过
		if tag == "-" {
			continue
		}

		if tag != "" {
			arr := strings.Split(tag, ",")
			key = arr[0]
		}

		if strings.Contains(tag, "omitempty") {
			zero := reflect.Zero(value.Type()).Interface()
			current := value.Interface()
			if reflect.DeepEqual(current, zero) {
				continue
			}
		}

		// Ignore field name of embedded struct
		if !field.Anonymous {
			// 如果非首个字段，则添加,
			if !first {
				sb.WriteRune(',')
			}
			sb.WriteString(`"`)
			sb.WriteString(key)
			sb.WriteString(`":`)
		}
		js.do(key, value.Interface(), field.Anonymous)
		first = false
	}
	if !embedded {
		sb.WriteRune('}')
	}
}

// Map stringify map
func (js *JSONStringify) Map(s interface{}) {
	sb := js.Sb
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	first := true
	sb.WriteRune('{')
	for _, index := range v.MapKeys() {
		value := v.MapIndex(index)
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		if isIgnore(value) {
			continue
		}
		if !first {
			sb.WriteRune(',')
		}
		first = false
		key := cast.ToString(index.Interface())
		sb.WriteString(`"`)
		sb.WriteString(key)
		sb.WriteString(`":`)
		js.do(key, value.Interface(), false)
	}
	sb.WriteRune('}')
}

// Array stringify array
func (js *JSONStringify) Array(s interface{}) {
	sb := js.Sb
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	len := v.Len()
	sb.WriteRune('[')
	first := true
	for i := 0; i < len; i++ {
		value := v.Index(i)
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		if isIgnore(value) {
			continue
		}

		if !first {
			sb.WriteRune(',')
		}
		first = false
		js.do(strconv.Itoa(i), value.Interface(), false)
	}
	sb.WriteRune(']')
}

func (js *JSONStringify) do(key string, s interface{}, embedded bool) {
	sb := js.Sb
	if js.Replacer != nil {
		replace, value := js.Replacer(key, s)
		if replace {
			sb.WriteString(value)
			return
		}
	}
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		js.St(s, embedded)
	case reflect.Map:
		js.Map(s)
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		js.Array(s)
	case reflect.String:
		sb.WriteRune('"')
		str := v.Interface().(string)
		sb.WriteString(strings.Replace(str, `"`, `\"`, -1))
		sb.WriteRune('"')
	default:
		sb.WriteString(cast.ToString(v.Interface()))
	}
}

// String json stringify
func (js *JSONStringify) String(s interface{}) string {
	js.do("", s, false)
	return js.Sb.String()
}

// String stringify
func String(s interface{}, replacer Replacer) string {
	js := jsPool.Get().(*JSONStringify)
	js.Sb.Reset()
	js.Replacer = replacer
	str := js.String(s)
	jsPool.Put(js)
	return str
}
