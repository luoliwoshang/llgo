#include <stdio.h>
#include <stdlib.h>
#include "cJSON.h"

// gcc mkjson.c -o create_json -I/opt/homebrew/Cellar/cjson/1.7.18/include/cjson -L/opt/homebrew/lib -lcjson

int main() {
    // 创建根对象
    cJSON *mod = cJSON_CreateObject();
    cJSON_AddItemToObject(mod,"name",cJSON_CreateString("math"));

    cJSON *syms  = cJSON_CreateArray();

    cJSON *fn = cJSON_CreateObject();
    cJSON_AddItemToObject(fn,"name",cJSON_CreateString("sqrt"));
    cJSON_AddItemToObject(fn,"sig",cJSON_CreateString("(x, /)"));
    cJSON_AddItemToArray(syms,fn);

    cJSON *v=cJSON_CreateObject();
    cJSON_AddItemToObject(v,"name",cJSON_CreateString("pi"));
    cJSON_AddItemToArray(syms,v);

    cJSON_AddItemToObject(mod,"items",syms);

    printf("%s\n", cJSON_PrintUnformatted(mod));
    
    cJSON_Delete(mod);
    // // 创建 items 数组
    // cJSON *items = cJSON_CreateArray();
    
    // // 创建第一个 item 对象
    // cJSON *item1 = cJSON_CreateObject();
    // cJSON_AddStringToObject(item1, "name", "sqrt");
    // cJSON_AddStringToObject(item1, "sig", "(x, /)");
    // cJSON_AddItemToArray(items, item1);

    // // 创建第二个 item 对象
    // cJSON *item2 = cJSON_CreateObject();
    // cJSON_AddStringToObject(item2, "name", "pi");
    // cJSON_AddItemToArray(items, item2);

    // // 将 items 数组添加到根对象
    // cJSON_AddItemToObject(root, "items", items);

    // // 将 JSON 对象转换为字符串
    // char *rendered = cJSON_Print(root);
    // printf("%s\n", cJSON_PrintUnformatted(root));

    // // 清理资源
    // cJSON_Delete(root);
    // free(rendered);

    return 0;
}
