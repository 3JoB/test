package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
)

const PNG_MAGIC = "\x89PNG\r\n\x1a\n"
var (
	IHDR = []byte("IHDR")
	PLTE = []byte("PLTE")
)

func fixupZip(data []byte, startOffset int64) {
	endCentralDirOffset := bytes.LastIndex(data, []byte("PK\x05\x06"))

	commentLength := len(data) - endCentralDirOffset - 22 + 0x10
	binary.LittleEndian.PutUint16(data[endCentralDirOffset+20:endCentralDirOffset+22], uint16(commentLength))

	cdentCount := binary.LittleEndian.Uint16(data[endCentralDirOffset+10 : endCentralDirOffset+12])

	centralDirStartOffset := binary.LittleEndian.Uint32(data[endCentralDirOffset+16 : endCentralDirOffset+20])
	binary.LittleEndian.PutUint32(data[endCentralDirOffset+16:endCentralDirOffset+20], uint32(int(centralDirStartOffset)+int(startOffset)))

	for i := 0; i < int(cdentCount); i++ {
		centralDirStartOffset = uint32(bytes.Index(data[centralDirStartOffset:], []byte("PK\x01\x02"))) + centralDirStartOffset

		off := binary.LittleEndian.Uint32(data[centralDirStartOffset+42 : centralDirStartOffset+46])
		binary.LittleEndian.PutUint32(data[centralDirStartOffset+42:centralDirStartOffset+46], uint32(int(off)+int(startOffset)))

		centralDirStartOffset++
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("USAGE: %s cover.png content.bin output.png\n", os.Args[0])
		os.Exit(1)
	}

	pngIn, _ := os.Open(os.Args[1])
	defer pngIn.Close()

	contentIn, _ := os.Open(os.Args[2])
	defer contentIn.Close()

	pngOut, _ := os.Create(os.Args[3])
	defer pngOut.Close()

	pngHeader := make([]byte, len(PNG_MAGIC))
	pngIn.Read(pngHeader)
	if string(pngHeader) != PNG_MAGIC {
		panic("Invalid PNG file")
	}
	pngOut.Write(pngHeader)

	var idatBody []byte
	var width, height int

	for {
		var chunkLen uint32
		binary.Read(pngIn, binary.BigEndian, &chunkLen)

		chunkType := make([]byte, 4)
		pngIn.Read(chunkType)

		chunkBody := make([]byte, chunkLen)
		pngIn.Read(chunkBody)

		var chunkCsum uint32
		binary.Read(pngIn, binary.BigEndian, &chunkCsum)

		if !bytes.Equal(chunkType, IHDR) && !bytes.Equal(chunkType, PLTE) && !bytes.Equal(chunkType, []byte("IDAT")) && !bytes.Equal(chunkType, []byte("IEND")) {
			fmt.Printf("Warning: dropping non-essential or unknown chunk: %s\n", string(chunkType))
			continue
		}

		if bytes.Equal(chunkType, IHDR) {
			width = int(binary.BigEndian.Uint32(chunkBody[0:4]))
			height = int(binary.BigEndian.Uint32(chunkBody[4:8]))
			fmt.Printf("Image size: %dx%dpx\n", width, height)
		}

		if bytes.Equal(chunkType, []byte("IDAT")) {
			idatBody = append(idatBody, chunkBody...)
			continue
		}

		if bytes.Equal(chunkType, []byte("IEND")) {
			startOffset, _ := pngOut.Seek(0, io.SeekCurrent)
			startOffset += int64(8 + len(idatBody))
			fmt.Printf("Embedded file starts at offset %x\n", startOffset)

			content, _ := io.ReadAll(contentIn)
			idatBody = append(idatBody, content...)

			if len(idatBody) > width*height {
				fmt.Println("ERROR: Input files too big for cover image resolution.")
				os.Exit(1)
			}

			if strings.ToLower(strings.Split(os.Args[2], ".")[1]) == "zip" || strings.ToLower(strings.Split(os.Args[2], ".")[1]) == "jar" {
				fmt.Println("Fixing up zip offsets...")
				fixupZip(idatBody, startOffset)
			}

			// write the IDAT chunk
			pngOut.Write([]byte{byte(len(idatBody) >> 24), byte(len(idatBody) >> 16), byte(len(idatBody) >> 8), byte(len(idatBody))})
			pngOut.Write([]byte("IDAT"))
			pngOut.Write(idatBody)
			crc := crc32.ChecksumIEEE(append([]byte("IDAT"), idatBody...))
			pngOut.Write([]byte{byte(crc >> 24), byte(crc >> 16), byte(crc >> 8), byte(crc)})

			// if we reached here, we're writing the IHDR, PLTE or IEND chunk
			pngOut.Write([]byte{byte(chunkLen >> 24), byte(chunkLen >> 16), byte(chunkLen >> 8), byte(chunkLen)})
			pngOut.Write(chunkType)
			pngOut.Write(chunkBody)
			crc = crc32.ChecksumIEEE(append(chunkType, chunkBody...))
			pngOut.Write([]byte{byte(crc >> 24), byte(crc >> 16), byte(crc >> 8), byte(crc)})

			if string(chunkType) == "IEND" {
				// we're done!
				break
			}

			// close our file handles
			pngIn.Close()
			contentIn.Close()
			pngOut.Close()
		}
	}
}
