#include <stdio.h>
#include <string.h>
#include <assert.h>
#include <zlib.h>

//  gcc -o decompress_example decompress_example.c -I/opt/homebrew/Cellar/zlib/1.3.1/include  -lz 
int main() {
    // 原始文本
    char text[] = "Hello, zlib compression!";
    unsigned long text_len = strlen(text) + 1;  // 包含终止符

    // 压缩数据缓冲区
    unsigned char compressed_data[100];
    unsigned long compressed_size = sizeof(compressed_data);

    // 输出uncompressed_data的size
    printf("Compressed size: %lu\n", compressed_size);

    // 压缩文本
    compress(compressed_data, &compressed_size, (const unsigned char*)text, text_len);

    // 输出压缩后的数据
    for (int i = 0; i < compressed_size; i++)
    {
        printf("%d",compressed_data[i]);
    }
    printf("\n");
    

    // 解压缩缓冲区
    unsigned char uncompressed_data[100];
    unsigned long uncompressed_size = sizeof(uncompressed_data);



    // 解压缩数据
    int result = uncompress(uncompressed_data, &uncompressed_size, compressed_data, compressed_size);
    if (result != Z_OK) {
        fprintf(stderr, "Failed to uncompress data: %d\n", result);
        return 1;
    }

    // 输出解压缩后的数据
    printf("Uncompressed data: %s\n", uncompressed_data);
    printf("Uncompressed size: %lu\n", uncompressed_size);

    return 0;
}
