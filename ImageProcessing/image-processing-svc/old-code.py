from flask import Flask, request, send_file, jsonify
from PIL import Image, ImageFilter
import os
import time
import zipfile

app = Flask(__name__)
TMP = "/tmp/"

# Ensure temp directory exists
os.makedirs(TMP, exist_ok=True)

def get_file_extension(file_name):
    """Ensure the file has a valid extension, default to PNG if missing."""
    file_ext = os.path.splitext(file_name)[1].lower()
    if not file_ext:
        file_ext = ".png"  # Default to PNG if no extension found
    return file_ext.lstrip(".")  # Remove leading dot for Pillow format compatibility

def flip(image, file_name):
    """Flip image in two directions and save."""
    path_list = []
    file_ext = get_file_extension(file_name)

    flip_lr_path = os.path.join(TMP, f"flip-left-right-{file_name}")
    flip_tb_path = os.path.join(TMP, f"flip-top-bottom-{file_name}")

    image.transpose(Image.FLIP_LEFT_RIGHT).save(flip_lr_path, format=file_ext.upper())
    image.transpose(Image.FLIP_TOP_BOTTOM).save(flip_tb_path, format=file_ext.upper())

    return [flip_lr_path, flip_tb_path]

def rotate(image, file_name):
    """Rotate image in three directions and save."""
    path_list = []
    file_ext = get_file_extension(file_name)

    for angle in [90, 180, 270]:
        path = os.path.join(TMP, f"rotate-{angle}-{file_name}")
        image.rotate(angle).save(path, format=file_ext.upper())
        path_list.append(path)
    return path_list

def filter_image(image, file_name):
    """Apply three different filters and save."""
    path_list = []
    file_ext = get_file_extension(file_name)

    filters = {
        "blur": ImageFilter.BLUR,
        "contour": ImageFilter.CONTOUR,
        "sharpen": ImageFilter.SHARPEN
    }
    for name, f in filters.items():
        path = os.path.join(TMP, f"{name}-{file_name}")
        image.filter(f).save(path, format=file_ext.upper())
        path_list.append(path)
    return path_list

def gray_scale(image, file_name):
    """Convert image to grayscale and save."""
    file_ext = get_file_extension(file_name)
    path = os.path.join(TMP, f"gray-scale-{file_name}")
    image.convert("L").save(path, format=file_ext.upper())
    return [path]

def resize(image, file_name):
    """Resize image to 128x128 and save."""
    file_ext = get_file_extension(file_name)
    path = os.path.join(TMP, f"resized-{file_name}")

    resized_image = image.copy()
    resized_image.thumbnail((128, 128))
    resized_image.save(path, format=file_ext.upper())
    return [path]

@app.route("/process", methods=["POST"])
def process_image():
    start_time = time.time()

    # Ensure file is uploaded
    if "file" not in request.files:
        return jsonify({"error": "No file uploaded"}), 400

    uploaded_file = request.files["file"]
    file_name = uploaded_file.filename.strip()
    
    # Ensure a valid file name
    if not file_name:
        return jsonify({"error": "Uploaded file has no name"}), 400

    file_ext = get_file_extension(file_name)
    file_name = f"{os.path.splitext(file_name)[0]}.{file_ext}"  # Ensure file has an extension

    file_path = os.path.join(TMP, file_name)
    uploaded_file.save(file_path)

    # Process Image
    processed_files = []
    try:
        with Image.open(file_path) as image:
            processed_files += flip(image, file_name)
            processed_files += rotate(image, file_name)
            processed_files += filter_image(image, file_name)
            processed_files += gray_scale(image, file_name)
            processed_files += resize(image, file_name)
    except Exception as e:
        return jsonify({"error": f"Image processing failed: {str(e)}"}), 500

    latency = time.time() - start_time

    # Zip all processed images
    zip_path = os.path.join(TMP, "processed_images.zip")
    with zipfile.ZipFile(zip_path, 'w') as zipf:
        for file in processed_files:
            zipf.write(file, os.path.basename(file))

    return send_file(zip_path, mimetype="application/zip", as_attachment=True, download_name="processed_images.zip")

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)

