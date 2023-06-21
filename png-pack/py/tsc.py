import sys 
import struct
import zlib

def embed_file(input_file, input_png, output_file):
    # Read input and png file
    with open(input_file, 'rb') as f:
        input_data = f.read()
    with open(input_png, 'rb') as f:
        png_data = f.read()

    # Find first IDAT chunk 
    first_idat = png_data.index(b'IDAT')

    # Get compression method 
    compression_method = png_data[first_idat+8]  

    # Find IDAT chunk 
    start = png_data.index(b'IDAT') 

    # Read IDAT header and length 
    idat_header = png_data[start:start+8]
    idat_len = struct.unpack('>I', png_data[start+8:start+12])[0]

    # Get IDAT data
    idat_data = png_data[start+12:start+12+idat_len]  

    # Decompress according to compression method
    decompressed_data = None
    if compression_method == 0:
        # No compression
        decompressed_data = idat_data  
    elif compression_method == 1:
        # zlib compression
        try:
            decompressed_data = zlib.decompress(idat_data)
        except zlib.error:
            # Not zlib data, try other method
            pass   

    # Build new IDAT chunk 
    new_idat = idat_header          
    new_idat += idat_data           # PNG image data

    # Add file length 
    input_len = len(input_data)
    new_idat += struct.pack('>I', input_len)   # File length     

    # Add file data
    new_idat += input_data         # File data

    # Replace IDAT chunk 
    png_data = png_data[:start] + new_idat + png_data[start+12+idat_len:]

    # Save as PNG
    with open(output_file, 'wb') as f:
        f.write(png_data)
        
if __name__ == '__main__':
    if len(sys.argv) != 4:
        print('Usage: {} <input file> <input png> <output png>')
        sys.exit(1)
        
    input_file = sys.argv[1]
    input_png = sys.argv[2]
    output_file = sys.argv[3]
    
    embed_file(input_file, input_png, output_file)
    print(f'File {input_file} embedded into {output_file}')