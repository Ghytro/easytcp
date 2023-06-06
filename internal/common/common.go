package common

import "golang.org/x/exp/constraints"

type KVStore map[string]interface{}

func (store KVStore) Get(key string) (interface{}, bool) {
	el, ok := store[key]
	return el, ok
}

func (store KVStore) MustGet(key string) interface{} {
	return store[key]
}

func (store KVStore) MustGetBool(key string) bool {
	el, ok := store[key]
	if !ok {
		return false
	}
	return el.(bool)
}

func (store KVStore) MustGetFloat(key string) float64 {
	el, ok := store[key]
	if !ok {
		return 0
	}
	return el.(float64)
}

func (e KVStore) MustGetStr(key string) string {
	result, ok := e[key]
	if !ok {
		return ""
	}
	return result.(string)
}

func (e KVStore) MustGetInt(key string) int {
	result, ok := e[key]
	if !ok {
		return 0
	}
	i, ok := result.(int)
	// это обусловлено тем, что linObjExtra на самом деле
	// парсится из жсона, interface{} не понимает int и парсит все как
	// float64, поэтому на всякий случай делаем вот такой фоллбек
	if !ok {
		f, ok := result.(float64)
		if !ok {
			return 0
		}
		return int(f)
	}
	return i
}

// MustGetIntSlice создает копию слайса для того чтобы получить нужного типа
func (e KVStore) MustGetIntSlice(key string) []int {
	result, ok := e[key]
	if !ok {
		return nil
	}
	resultSlice, ok := result.([]interface{})
	if !ok {
		return nil
	}
	result = make([]int, len(resultSlice))
	resInt := result.([]int)
	for i, el := range resultSlice {
		resInt[i], ok = el.(int)
		if !ok {
			// float64 из за парсинга жсона
			f, ok := el.(float64)
			if !ok {
				return nil
			}
			resInt[i] = int(f)
		}
	}
	return resInt
}

func (e KVStore) MustGetInterfaceMap(key string) map[string]interface{} {
	result, ok := e[key]
	if !ok {
		return nil
	}
	return result.(map[string]interface{})
}

func (e KVStore) MustGetInterfaceSlice(key string) []interface{} {
	result, ok := e[key]
	if !ok {
		return nil
	}
	return result.([]interface{})
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
