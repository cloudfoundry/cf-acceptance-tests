package testhelpers

import "reflect"

func AlwaysReturn(receiver interface{}, args ...interface{}) {
	validReceiver(receiver)
	validArgs(receiver, args)

	go write(receiver, args)
}

func write(receiver interface{}, args []interface{}) {
	val := reflect.ValueOf(receiver)
	if val.Kind() == reflect.Chan {
		go writeToChannel(receiver, args[0])
		return
	}

	if val.Kind() == reflect.Struct {
		writeToStruct(receiver, args)
		return
	}
}

func writeToStruct(receiver interface{}, args []interface{}) {
	value := reflect.ValueOf(receiver)
	for i, arg := range args {
		argVal := reflect.ValueOf(arg)
		chVal := value.Field(i)
		go doChanWrite(chVal, argVal)
	}
}

func writeToChannel(ch interface{}, arg interface{}) {
	val := reflect.ValueOf(ch)
	argVal := reflect.ValueOf(arg)
	go doChanWrite(val, argVal)
}

func doChanWrite(valCh, argVal reflect.Value) {
	chType := valCh.Type().Elem()
	if argVal.Type().ConvertibleTo(chType) {
		argVal = argVal.Convert(chType)
	}

	for {
		valCh.Send(argVal)
	}
}

func validReceiver(receiver interface{}) {
	val := reflect.ValueOf(receiver)
	if val.Kind() == reflect.Chan {
		validChannelReceiver(receiver)
		return
	}

	if val.Kind() == reflect.Struct {
		validStructReceiver(receiver)
		return
	}

	panic("AlwaysReturn requires a send only channel or struct full of channel fields")
}

func validChannelReceiver(receiver interface{}) {
	if reflect.TypeOf(receiver).ChanDir() == reflect.RecvDir {
		panic("AlwaysReturn requires a send only channel or struct full of channel fields")
	}
}

func validStructReceiver(receiver interface{}) {
	structType := reflect.TypeOf(receiver)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Type.Kind() != reflect.Chan || field.Type.ChanDir() == reflect.RecvDir {
			panic("AlwaysReturn requires a send only channel or struct full of channel fields")
		}
	}
}

func validArgs(receiver interface{}, args []interface{}) {
	if len(args) == 0 {
		panic("args must have a length greater than 0")
	}

	val := reflect.ValueOf(receiver)
	if val.Kind() == reflect.Chan {
		validChannelArgs(receiver, args)
		return
	}

	if val.Kind() == reflect.Struct {
		validStructArgs(receiver, args)
		return
	}
}

func validChannelArgs(receiver interface{}, args []interface{}) {
	if len(args) != 1 {
		panic("a channel can only send a single argument")
	}

	argType := reflect.TypeOf(args[0])
	chType := reflect.TypeOf(receiver)
	if !argType.AssignableTo(chType.Elem()) && !argType.ConvertibleTo(chType.Elem()) {
		panic("channel type and argument type have to match")
	}
}

func validStructArgs(receiver interface{}, args []interface{}) {
	structType := reflect.TypeOf(receiver)
	if len(args) != structType.NumField() {
		panic("a struct requires the same number of arguments as fields")
	}

	for i, arg := range args {
		argType := reflect.TypeOf(arg)
		chType := structType.Field(i).Type.Elem()

		if !argType.AssignableTo(chType) && !argType.ConvertibleTo(chType) {
			panic("a struct requires the same type for each arguments")
		}
	}
}
