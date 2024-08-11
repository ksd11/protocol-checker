package main

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/envoyproxy/protoc-gen-validate/templates/shared"
	"github.com/envoyproxy/protoc-gen-validate/validate"
	pgs "github.com/lyft/protoc-gen-star/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var debug_is_required bool = false
var debug_field_type string = ""
var debug_field_name string = ""
var debug_field_value string = ""
var debug_ignore_empty bool = false
var debug_rules map[string]interface{} = make(map[string]interface{})

func debug() {
	res := ""
	if debug_is_required {
		res += "required "
	} else {
		res += "optional "
	}
	res += debug_field_type + " " + debug_field_name
	if debug_ignore_empty {
		res += " = \"\" [ignore_empty]"
		fmt.Fprintln(os.Stderr, res)
		return
	}

	res += " = " + debug_field_value + " ["
	for k, v := range debug_rules {
		res += k + "=" + fmt.Sprintf("%v", v) + ","
	}
	res += "]"
	fmt.Fprintln(os.Stderr, res)
}

func debug_clear() {
	debug_is_required = false
	debug_field_type = ""
	debug_field_name = ""
	debug_field_value = ""
	debug_ignore_empty = false
	debug_rules = make(map[string]interface{})
}

/*
*

	检测字段是否复合规范
	1. 若字段是必须的，是否已经设置
	2. 字段的类型是否一致
	3. 字段是否符合校验规则
*/
func ParseField(f pgs.Field, rawData map[string]string) (isValidate bool, msg []string) {
	debug_clear() // for debug
	debug_field_name = f.Name().String()
	debug_field_type = f.Type().ProtoType().String()

	isValidate = true
	skip := false // 是否跳过校验，比如：可选字段未设置

	// 检验必要字段是否已经设置
	isValidate, msg, skip = checkRequired(f, rawData)
	if !isValidate || skip {
		debug() // for debug
		return
	}

	// 检验字段类型和校验信息
	isValidate, msg = checkRule(f, rawData)

	debug() // for debug
	return
}

func checkRequired(f pgs.Field, rawData map[string]string) (isValidate bool, msg []string, skip bool) {
	isValidate = true
	msg = []string{}

	if f.Required() {
		// 检测必要字段是否已经设置
		debug_is_required = true // for debug
		if _, ok := rawData[f.Name().String()]; !ok {
			isValidate = false
			msg = append(msg, fmt.Sprintf("字段 %s 是必须的", f.Name().String()))
		}
	} else {
		// 可选字段不存在
		debug_is_required = false // for debug
		if _, ok := rawData[f.Name().String()]; !ok {
			skip = true
		}
	}
	return
}

/*
*

	处理数值类型的数据
	1. 先获取Number的所有校验规则
	2. 然后验证Number对应的value是否符合校验规则
*/
func handleNumber[T Number](value_any any, rules protoreflect.ProtoMessage) (isValidate bool, msg []string) {
	parsedRules := parseNumber[T](rules)
	return validateRules[T](value_any.(T), parsedRules)
}

func handleBool(value_any any, bool_rules protoreflect.ProtoMessage) (isValidate bool, msg []string) {
	val := getValue(bool_rules)
	var rules []RuleFunc[bool]

	rules = addRule[bool, bool]("Const", ScalarConst)(val, rules)
	return validateRules[bool](value_any.(bool), rules)
}

func handleString(value_any any, bool_rules protoreflect.ProtoMessage) (isValidate bool, msg []string) {
	val := getValue(bool_rules)
	var rules []RuleFunc[string]

	rules = addRule[string, string]("Const", ScalarConst)(val, rules)
	rules = addRule[string, uint64]("Len", StringLen)(val, rules)
	rules = addRule[string, uint64]("MinLen", StringMinLen)(val, rules)
	rules = addRule[string, uint64]("MaxLen", StringMaxLen)(val, rules)
	rules = addRule[string, uint64]("LenBytes", StringLenBytes)(val, rules)
	rules = addRule[string, uint64]("MinBytes", StringMinBytes)(val, rules)
	rules = addRule[string, uint64]("MaxBytes", StringMaxBytes)(val, rules)
	rules = addRule[string, string]("Pattern", StringPattern)(val, rules)
	rules = addRule[string, string]("Prefix", StringPrefix)(val, rules)
	rules = addRule[string, string]("Suffix", StringSuffix)(val, rules)
	rules = addRule[string, string]("Contains", StringContains)(val, rules)
	rules = addRule[string, string]("NotContains", StringNotContains)(val, rules)
	rules = addInRule(val, rules)
	rules = addNotInRule(val, rules)
	return validateRules(value_any.(string), rules)
}

func handleBytes(value_any any, bool_rules protoreflect.ProtoMessage) (isValidate bool, msg []string) {
	isValidate = false
	msg = append(msg, "bytes类型校验暂不支持")
	return
}

func checkRule(f pgs.Field, rawData map[string]string) (isValidate bool, msg []string) {
	isValidate = true
	msg = []string{}
	ruleContext, err := rulesContext(f)
	if err != nil {
		isValidate = false
		msg = append(msg, err.Error())
		return
	} else if reflect.ValueOf(ruleContext.Rules).IsNil() {
		// https://www.cnblogs.com/mfrank/p/16831877.html 不能直接写成Nil比较
		// 无validate校验，跳过。但是仍需要校验类型
		_, err := TypeConvertFuncMap[ruleContext.Typ](rawData[f.Name().String()])
		if err != nil {
			isValidate = false
			msg = append(msg, err.Error())
			return
		}
		debug_field_value = fmt.Sprintf("%v", rawData[f.Name().String()])
		return

	}

	// ignore_empty
	reflect_val := getValue(ruleContext.Rules)
	ok, ignore_empty := GetBool(reflect_val, "IgnoreEmpty")
	if ok && ignore_empty && rawData[f.Name().String()] == "" {
		ignore_empty = true
		debug_ignore_empty = true // for debug
		return
	}

	// 校验类型
	value_any, err := TypeConvertFuncMap[ruleContext.Typ](rawData[f.Name().String()])
	if err != nil {
		isValidate = false
		msg = append(msg, err.Error())
		return
	}
	debug_field_value = fmt.Sprintf("%v", rawData[f.Name().String()])

	// validate
	switch ruleContext.Typ {
	case "uint32", "fixed32":
		return handleNumber[uint32](value_any, ruleContext.Rules)
	case "uint64", "fixed64":
		return handleNumber[uint64](value_any, ruleContext.Rules)
	case "int32", "sint32", "sfixed32":
		return handleNumber[int32](value_any, ruleContext.Rules)
	case "int64", "sint64", "sfixed64":
		return handleNumber[int64](value_any, ruleContext.Rules)
	case "double":
		return handleNumber[float64](value_any, ruleContext.Rules)
	case "float":
		return handleNumber[float32](value_any, ruleContext.Rules)
	case "bool":
		return handleBool(value_any, ruleContext.Rules)
	case "string":
		return handleString(value_any, ruleContext.Rules)
	case "bytes":
		return handleBytes(value_any, ruleContext.Rules)
	default:
		isValidate = false
		msg = append(msg, fmt.Sprintf("不支持类型 %s", ruleContext.Typ))
	}

	return
}

func getValue(numberRules protoreflect.ProtoMessage) reflect.Value {
	val := reflect.ValueOf(numberRules)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}

// 返回一堆验证函数
func parseNumber[T Number](numberRules protoreflect.ProtoMessage) []RuleFunc[T] {
	val := getValue(numberRules)
	var rules []RuleFunc[T]

	rules = addRule[T, T]("Const", ScalarConst)(val, rules)
	rules = addRule[T, T]("Lt", NumberLt)(val, rules)
	rules = addRule[T, T]("Lte", NumberLte)(val, rules)
	rules = addRule[T, T]("Gt", NumberGt)(val, rules)
	rules = addRule[T, T]("Gte", NumberGte)(val, rules)
	rules = addInRule(val, rules)
	rules = addNotInRule(val, rules)

	return rules
}

// 验证value是否满足规则，只要有任意一个规则不通过，则不通过
func validateRules[T any](val T, rules []RuleFunc[T]) (check bool, msg []string) {
	check = true
	msg = []string{}
	for _, rule := range rules {
		if ok, m := rule(val); !ok {
			msg = append(msg, m)
			check = false
		}
	}
	return
}

func rulesContext(f pgs.Field) (out shared.RuleContext, err error) {
	out.Field = f

	var rules validate.FieldRules
	if _, err = f.Extension(validate.E_Rules, &rules); err != nil {
		return
	}

	var wrapped bool
	if out.Typ, out.Rules, out.MessageRules, wrapped = resolveRules(f.Type(), &rules); wrapped {
		out.WrapperTyp = out.Typ
		out.Typ = "wrapper"
	}

	if out.Typ == "error" {
		err = fmt.Errorf("unknown rule type (%T)", rules.Type)
	}

	return
}

func resolveRules(typ interface{ IsEmbed() bool }, rules *validate.FieldRules) (ruleType string, rule proto.Message, messageRule *validate.MessageRules, wrapped bool) {
	fmt.Fprintln(os.Stderr, typ.(pgs.FieldType).ProtoType().String())
	// r := rules.GetType().(type)
	switch typ.(pgs.FieldType).ProtoType() {
	case pgs.FloatT:
		ruleType, rule, wrapped = "float", rules.GetFloat(), typ.IsEmbed()
	case pgs.DoubleT:
		ruleType, rule, wrapped = "double", rules.GetDouble(), typ.IsEmbed()
	case pgs.Int32T:
		ruleType, rule, wrapped = "int32", rules.GetInt32(), typ.IsEmbed()
	case pgs.Int64T:
		ruleType, rule, wrapped = "int64", rules.GetInt64(), typ.IsEmbed()
	case pgs.UInt32T:
		ruleType, rule, wrapped = "uint32", rules.GetUint32(), typ.IsEmbed()
	case pgs.UInt64T:
		ruleType, rule, wrapped = "uint64", rules.GetUint64(), typ.IsEmbed()
	case pgs.SInt32:
		ruleType, rule, wrapped = "sint32", rules.GetSint32(), false
	case pgs.SInt64:
		ruleType, rule, wrapped = "sint64", rules.GetSint64(), false
	case pgs.Fixed32T:
		ruleType, rule, wrapped = "fixed32", rules.GetFixed32(), false
	case pgs.Fixed64T:
		ruleType, rule, wrapped = "fixed64", rules.GetFixed64(), false
	case pgs.SFixed32:
		ruleType, rule, wrapped = "sfixed32", rules.GetSfixed32(), false
	case pgs.SFixed64:
		ruleType, rule, wrapped = "sfixed64", rules.GetSfixed64(), false
	case pgs.BoolT:
		ruleType, rule, wrapped = "bool", rules.GetBool(), typ.IsEmbed()
	case pgs.StringT:
		ruleType, rule, wrapped = "string", rules.GetString_(), typ.IsEmbed()
	case pgs.BytesT:
		ruleType, rule, wrapped = "bytes", rules.GetBytes(), typ.IsEmbed()
	case pgs.EnumT:
		ruleType, rule, wrapped = "enum", rules.GetEnum(), false
		// case *validate.FieldRules_Repeated:
		// 	ruleType, rule, wrapped = "repeated", r.Repeated, false
		// case *validate.FieldRules_Map:
		// 	ruleType, rule, wrapped = "map", r.Map, false
		// case *validate.FieldRules_Any:
		// 	ruleType, rule, wrapped = "any", r.Any, false
		// case *validate.FieldRules_Duration:
		// 	ruleType, rule, wrapped = "duration", r.Duration, false
		// case *validate.FieldRules_Timestamp:
		// 	ruleType, rule, wrapped = "timestamp", r.Timestamp, false
	_:
		if ft, ok := typ.(pgs.FieldType); ok && ft.IsRepeated() {
			return "repeated", &validate.RepeatedRules{}, rules.Message, false
		} else if ok && ft.IsMap() && ft.Element().IsEmbed() {
			return "map", &validate.MapRules{}, rules.Message, false
		} else if typ.IsEmbed() {
			return "message", rules.GetMessage(), rules.GetMessage(), false
		}
		return "none", nil, nil, false
	default:
		ruleType, rule, wrapped = "error", nil, false
	}

	return ruleType, rule, rules.Message, wrapped
}

// add Rules

// var addRulesFuncMap map[string]func(reflect.Value, []RuleFunc[any]) []RuleFunc[any] = make(map[string]func(reflect.Value, []RuleFunc[any]) []RuleFunc[any])

// func init() {
// 	addRulesFuncMap["Const"] =
// }

// AbcDef -> abc_def
func camelCaseToSnakeCase(s string) string {
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}

// 添加"校验规则函数"的函数，T代表校验的类型
type AddRuleFunc[T any] func(reflect.Value, []RuleFunc[T]) []RuleFunc[T]

// addRule("Const", ScalarConst(constVal)) -> AddRuleFunc[T]
// T: 校验类型
// V: 字段类型
func addRule[T any, V any](name string, rule_func_getter RuleFuncGetter[T, V]) AddRuleFunc[T] {
	sname := camelCaseToSnakeCase(name) // for debug的输出名
	return func(rval reflect.Value, rules []RuleFunc[T]) []RuleFunc[T] {
		ok, val := GetFieldPointer[V](rval, name)
		if ok {
			debug_rules[sname] = val // for debug
			return append(rules, rule_func_getter(val))
		}
		return rules
	}
}

func addInRule[T Number | string](val reflect.Value, rules []RuleFunc[T]) []RuleFunc[T] {
	ok, in := GetFieldArray[T](val, "In")
	if ok {
		debug_rules["in"] = in // for debug
		return append(rules, ScalarIn(in))
	}
	return rules
}

func addNotInRule[T Number | string](val reflect.Value, rules []RuleFunc[T]) []RuleFunc[T] {
	ok, not_in := GetFieldArray[T](val, "NotIn")
	if ok {
		debug_rules["not_in"] = not_in // for debug
		rules = append(rules, ScalarNotIn(not_in))
	}
	return rules
}
