package render

import (
	"fmt"
	"strings"

	j "github.com/jsonnet-libs/k8s/pkg/builder"
	"github.com/jsonnet-libs/k8s/pkg/model"
	"github.com/jsonnet-libs/k8s/pkg/swagger"
)

const (
	LocalApiVersion = "_apiVersion"
)

func Modifier(name string, i interface{}) []j.Type {
	switch t := i.(type) {
	case model.Modifier:
		return modFunction(name, t)
	case model.Object:
		o := modObject(name, t)
		return []j.Type{j.Comment(o, t.Help)}
	}
	panic(fmt.Sprintf("unexpected %T", i))
}

func modObject(name string, o model.Object) j.ObjectType {
	childs := make([]j.Type, 0, len(o.Fields))
	for k, m := range o.Fields {
		childs = append(childs, Modifier(k, m)...)
	}

	SortFields(childs)

	newObj := j.Object
	if len(childs) == 1 && !isFuncType(childs[0]) {
		newObj = j.ConciseObject
	}

	return newObj(name, childs...)
}

func modFunction(name string, f model.Modifier) []j.Type {
	// parameters
	args := j.Args(
		j.Required(j.String(f.Arg.Key, "")),
	)

	out := make([]j.Type, 0, 2)

	// withXyz()
	set := newFn(f, false)
	out = append(out, j.Func(name, args, j.ConciseObject("", set)))

	// withXyzMixin()
	if f.Type == swagger.TypeObject || f.Type == swagger.TypeArray {
		add := newFn(f, true)
		out = append(out, j.Func(name+"Mixin", args, j.ConciseObject("", add)))
	}

	return out
}

func newFn(f model.Modifier, adder bool) j.Type {
	elems := strings.Split(f.Target, ".")
	ret := reduceReverse(elems, func(i int, s string, o j.Type) j.Type {
		switch i {
		case 0:
			return j.Ref(s, f.Arg.Key)
		case 1:
			if !adder {
				return j.ConciseObject(s, o)
			}
			fallthrough
		default:
			return j.ConciseObject(s, j.Merge(o))
		}
	})
	if len(elems) != 1 || adder {
		ret = j.Merge(ret)
	}

	return ret
}

// reduceReverse calls f for each in arr in reverse order.
// o will the result of the previous element's invocation, nil if i==0
func reduceReverse(arr []string, f func(i int, s string, o j.Type) j.Type) j.Type {
	size := len(arr) - 1
	var last j.Type
	for ii := range arr {
		i := size - ii
		last = f(ii, arr[i], last)
	}
	return last
}

func isFuncType(t j.Type) bool {
	_, ok := t.(j.FuncType)
	return ok
}
