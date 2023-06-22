package pngtool

import (
    "bytes"
    "encoding/binary"
    "errors"
    "hash/crc32"
    "io"
    "os"
)

const PNG_SIGNATURE = "\x89\x50\x4E\x47\x0D\x0A\x1A\x0A" 

type Chunk struct {
    Length uint32
    Type   [4]byte
    Data   []byte
    CRC    uint32
}

func (c *Chunk) Write(w io.Writer) error {
    if err := binary.Write(w, binary.BigEndian, c.Length); err != nil {
        return err
    }
    if _, err := w.Write(c.Type[:]); err != nil {
        return err
    }
    if _, err := w.Write(c.Data); err != nil {
        return err
    }
    return binary.Write(w, binary.BigEndian, c.CRC)
}

func ParseChunk(r io.Reader) (*Chunk, error) {
    var length uint32
    if err := binary.Read(r, binary.BigEndian, &length); err != nil {
        return nil, err
    }

    typeArr := make([]byte, 4)
    if _, err := io.ReadFull(r, typeArr); err != nil {
        return nil, err
    }

    data := make([]byte, length)
    if _, err := io.ReadFull(r, data); err != nil {
        return nil, err
    }

    var crc uint32
    if err := binary.Read(r, binary.BigEndian, &crc); err != nil {
        return nil, err
    }

    return &Chunk{
        Length: length,
        Type:   [4]byte(typeArr),
        Data:   data,
        CRC:    crc,
    }, nil
}

type PNG struct {
    Width      uint32
    Height     uint32
    IDATStart  int64
    IDATEnd    int64
    Data       []byte 
	EmbedChunk *Chunk
}

func NewPNG(r io.Reader) (*PNG, error) {
    // Check PNG signature
    signature := make([]byte, 8)
    if _, err := io.ReadFull(r, signature); err != nil {
        return nil, err
    } else if !bytes.Equal(signature, []byte(PNG_SIGNATURE)) {
        return nil, errors.New("invalid PNG signature")
    }

    png := new(PNG)

    for {
        chunk, err := ParseChunk(r)
        if err != nil {
            return nil, err
        }

        if bytes.Equal(chunk.Type[:], []byte("IHDR")) {
            // Get image info
            buf := bytes.NewBuffer(chunk.Data)
            if err := binary.Read(buf, binary.BigEndian, &png.Width); err != nil {
                return nil, err
            }
            if err := binary.Read(buf, binary.BigEndian, &png.Height); err != nil {
                return nil, err
            }
        } else if bytes.Equal(chunk.Type[:], []byte("IDAT")) {
            // Record first IDAT offset
            if png.IDATStart == 0 {
                png.IDATStart = int64(chunk.Length) + 12 // chunk len(4) + type(4) + crc(4)
            }
            png.IDATEnd = int64(chunk.Length) + png.IDATEnd + 12 

            png.Data = append(png.Data, chunk.Data...)
        } else if bytes.Equal(chunk.Type[:], []byte("IEND")) {
            break
        }
    }

    return png, nil
}

func (png *PNG) Embed(target io.Reader, output *os.File) error {
    // Get target file data 
    data, err := io.ReadAll(target)
    if err != nil {
        return err
    }
    
    // Build new IDAT chunk with file data
    embedChunk := &Chunk{
        Type:   [4]byte{'I', 'D', 'A', 'T'},
        Data:   data,
        CRC:    crc32.ChecksumIEEE(append([]byte("IDAT"), data...)),
    }
    
    // Save embed chunk info 
    png.EmbedChunk = embedChunk
    
    var buf bytes.Buffer
    
    // Write PNG header
    buf.WriteString(PNG_SIGNATURE)
    
    // Write PNG chunks before first IDAT 
    for _, c := range png.Chunks {
        if c.Type == "IDAT" {
            break
        }
        if err := c.Write(&buf); err != nil {
            return err
        }
    }
    
    // Write new IDAT chunk
    if err := embedChunk.Write(&buf); err != nil {
        return err
    }
    
    // Write rest IDAT and IEND chunks 
    for _, c := range png.Chunks {
        if c.Type == "IDAT" || c.Type == "IEND" {
            if err := c.Write(&buf); err != nil {
                return err
            }
        }
    }
    
    // Save new PNG data
    pngData := buf.Bytes() 
    if _, err := output.Write(pngData); err != nil {
        return err
    }
    
    return nil 
}

func (png *PNG) UnEmbed(target *os.File) (*os.File, error) {
    // ... 
}