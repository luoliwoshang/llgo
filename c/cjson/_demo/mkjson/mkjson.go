package main

import (
	"github.com/goplus/llgo/c"
	"github.com/goplus/llgo/c/cjson"
)

func main() {
	mod := cjson.Object()
	// cJSON *mod = cJSON_CreateObject();
	mod.SetItem(c.Str("name"), cjson.String(c.Str("math")))
	// cJSON_AddItemToObject(mod, "name", cJSON_CreateString("math"))

	syms := cjson.Array()
	// cJSON *syms  = cJSON_CreateArray();

	fn := cjson.Object()
	// cJSON *fn = cJSON_CreateObject();
	fn.SetItem(c.Str("name"), cjson.String(c.Str("sqrt")))
	// cJSON_AddItemToObject(fn,"name",cJSON_CreateString("sqrt"));
	fn.SetItem(c.Str("sig"), cjson.String(c.Str("(x, /)")))
	// cJSON_AddItemToObject(fn,"sig",cJSON_CreateString("(x, /)"));
	syms.AddItem(fn)
	// cJSON_AddItemToArray(syms, fn)

	v := cjson.Object()
	// cJSON *v=cJSON_CreateObject();

	v.SetItem(c.Str("name"), cjson.String(c.Str("pi")))
	// cJSON_AddItemToObject(v,"name",cJSON_CreateString("pi"));
	syms.AddItem(v)
	// cJSON_AddItemToArray(syms,v);

	mod.SetItem(c.Str("items"), syms)
	// cJSON_AddItemToObject(mod,"items",syms);

	c.Printf(c.Str("%s\n"), mod.CStr())
	// printf("%s\n", cJSON_PrintUnformatted(mod))

	mod.Delete()
}
