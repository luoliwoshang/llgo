package main

import (
	"unsafe"

	"github.com/goplus/llgo/c"
	"github.com/goplus/llgo/c/zlib"
)

func main() {
	text := c.Str("Hello, zlib compression!") // char text[] = "Hello, zlib compression!";
	unit8_text := (*uint8)(unsafe.Pointer(text))
	text_len := uint32(c.Strlen(text) + 1) // unsigned long text_len = strlen(text) + 1;  // 包含终止符
	c.Printf(c.Str("text_len = %d\n"), text_len)

	// unsigned char compressed_data[100];
	// unsigned long compressed_size = sizeof(compressed_data);
	var compressed_data [100]uint8
	uncompressedSize := uint32(len(compressed_data)) // unsigned long

	zlib.Uncompress(&compressed_data[0], &uncompressedSize, unit8_text, text_len)

	// 输出压缩后的字节数组
	for i := 0; i < int(uncompressedSize); i++ {
		c.Printf(c.Str("%d "), compressed_data[i])
	}
}
