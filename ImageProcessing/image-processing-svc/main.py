from flask import Flask, request, jsonify
from cloudevents.http import from_http
from PIL import Image, ImageFilter
import os
import requests
import time
import io

app = Flask(__name__)
TMP = "/tmp/"
os.makedirs(TMP, exist_ok=True)

def get_file_extension(file_name):
    """Ensure the file has a valid extension, default to PNG if missing."""
    file_ext = os.path.splitext(file_name)[1].lower()
    return file_ext.lstrip(".") if file_ext else "png"

def flip(image):
    """Flip image in two directions."""
    image.transpose(Image.FLIP_LEFT_RIGHT)
    image.transpose(Image.FLIP_TOP_BOTTOM)

def rotate(image):
    """Rotate image in three directions."""
    for angle in [90, 180, 270]:
        image.rotate(angle)

def filter_image(image):
    """Apply multiple filters."""
    image.filter(ImageFilter.BLUR)
    image.filter(ImageFilter.CONTOUR)
    image.filter(ImageFilter.SHARPEN)

def gray_scale(image):
    """Convert image to grayscale."""
    image.convert("L")

def resize(image):
    """Resize image to 128x128."""
    image.thumbnail((128, 128))

@app.route("/", methods=["POST"])
def receive_event():
    try:
        start_time = time.time()

        # Parse CloudEvent from the request
        event = from_http(request.headers, request.get_data())

        # Extract the image URL
        image_url = event.data.get("image_url")
        if not image_url:
            return jsonify({"error": "Missing image_url in event"}), 400

        # Download the image
        response = requests.get(image_url, stream=True)
        if response.status_code != 200:
            return jsonify({"error": "Failed to download image"}), 500

        # Load image into memory
        file_name = image_url.split("/")[-1]  # Extract filename from URL
        file_ext = get_file_extension(file_name)
        image = Image.open(io.BytesIO(response.content))

        # Perform CPU-heavy processing
        flip(image)
        rotate(image)
        filter_image(image)
        gray_scale(image)
        resize(image)

        latency = time.time() - start_time

        # Return metadata, **no need to save the processed images**
        return jsonify({
            "status": "success",
            "processing_time": round(latency, 3),
            "service": "image-processing",
            "event_id": event["id"],
            "original_filename": file_name
        }), 200

    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
