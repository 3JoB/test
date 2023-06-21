import sys
import os
import struct
import zlib

PNG_MAGIC = b'\x89PNG\r\n\x1a\n'

class Chunk:
    def __init__(self, start, end, length, chunk_type, data):
        self.start = start
        self.end = end
        self.length = length
        self.type = chunk_type
        self.data = data
        
class PNG:
    def __init__(self, input_png):
        with open(input_png, 'rb') as f:
            self.png_data = f.read()
            
        # Check PNG header
        if self.png_data[:8] != PNG_MAGIC:
            raise ValueError('Input file is not a PNG image.')
            
        # Get image info from IHDR 
        self.ihdr_start = self.png_data.index(b'IHDR')
        self.width, self.height = struct.unpack('>II', self.png_data[self.ihdr_start+8:self.ihdr_start+16])
        
    def embed_file(self, input_file, output_file):
        # Find and record IDAT chunk info 
        idat_start, idat_end = None, None
        idat_data = b''
        for chunk in self.parse_chunks():
            if chunk.type == b'IDAT':
                if not idat_start:
                    idat_start = chunk.start 
                idat_end = chunk.end
                idat_data += chunk.data
        
        # Get file data
        file_data = open(input_file, 'rb').read() 
        
        # Build new IDAT with file data
        file_chunk = b'IDAT' + struct.pack('>I', len(file_data))
        file_chunk += file_data
        file_chunk += struct.pack('>I', zlib.crc32(b'IDAT' + file_data))
        
        # Replace original IDAT, fix metadata
        new_png_data = self.png_data[:idat_start] + file_chunk + self.png_data[idat_end:]
        new_png_data = self.fix_metadata(new_png_data, idat_start, idat_end, len(file_chunk))
        
        # Save new PNG and check 
        with open(output_file, 'wb') as f:
            f.write(new_png_data)
        self.png_check(output_file)
        
        print(f'File {input_file} embedded into {output_file}')
        
    def parse_chunks(self):
        chunks = []
        while True:
            # Get chunk info
            chunk_start = self.png_data.index(b'\x00\x00\x00\x00')
            chunk_len = int.from_bytes(self.png_data[chunk_start:chunk_start+4], 'big')
            chunk_type = self.png_data[chunk_start+4:chunk_start+8]
            chunk_end = chunk_start + chunk_len + 12
            
            chunks.append(Chunk(chunk_start, chunk_end, chunk_len, chunk_type, self.png_data[chunk_start+8:chunk_end]))           
            
            if chunk_type == b'IEND':
                break
                
            self.png_data = self.png_data[chunk_end:]
            
        return chunks  
    
    def fix_metadata(self, png_data, idat_start, idat_end, file_chunk_len):
        if idat_start is None:
            return png_data
        
        # Fix CRC     
        prev_chunk = None 

        for chunk in self.parse_chunks():   
            if chunk.start < idat_start:
                png_data = png_data[:chunk.end] + png_data[idat_end:]
                prev_chunk = chunk
                continue         
            if prev_chunk:
                offset = chunk.start - prev_chunk.end 
                chunk.start -= offset
                chunk.end -= offset
            crc = zlib.crc32(png_data[chunk.start:chunk.end])
            png_data = png_data[:chunk.end] + struct.pack('>I', crc) + png_data[chunk.end + 4:] 
            prev_chunk = chunk
            
        # Fix lengths   
        for i in range(0, png_data.index(b'IEND'), 12):
            chunk_len = int.from_bytes(png_data[i:i+4], 'big')
            png_data = png_data[:i] + struct.pack('>I', chunk_len + file_chunk_len) + png_data[i+4:]
            
        return png_data
    
    def png_check(self, png_file):
        pass

if __name__ == '__main__':
    input_file = sys.argv[1]
    input_png = sys.argv[2]
    output_file = sys.argv[3]
    
    png = PNG(input_png)
    png.embed_file(input_file, output_file)