#This is an example that uses the websockets api and the SaveImageWebsocket node to get images directly without
#them being saved to disk

import os
import websocket
import uuid
import json
import urllib.request
from enum import StrEnum
import random

ASSETS_FOLDER = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SERVER_ADDRESS = "127.0.0.1:8000"

INPUT_NODE_WORKFLOW_ID = "6"
OUTPUT_NODE_WORKFLOW_ID = "11"
SEED_NODE_WORKFLOW_ID = "3"
STEPS_NODE_WORKFLOW_ID = "3"
IMAGE_SIZE_NODE_WORKFLOW_ID = "5"

class ImageRatio(StrEnum):
    SQUARE = "square"
    LANDSCAPE = "landscape"
    PORTRAIT = "portrait"

client_id = str(uuid.uuid4())
content_prompt = "A beautiful space station in the sky, seen from the ground"
style_prompt = "Unrealistic, 3D render, pixel, starcraft 1 artstyle"
workflow_name = "default.json"
seed = random.randint(0, 1000000)
steps = 20
ratio = ImageRatio.LANDSCAPE


def queue_prompt(prompt):
    p = {"prompt": prompt, "client_id": client_id}
    data = json.dumps(p).encode('utf-8')
    req =  urllib.request.Request("http://{}/prompt".format(SERVER_ADDRESS), data=data)
    return json.loads(urllib.request.urlopen(req).read())

def get_images(ws, prompt):
    prompt_id = queue_prompt(prompt)['prompt_id']
    output_images = {}
    current_node = ""
    while True:
        out = ws.recv()
        if isinstance(out, str):
            message = json.loads(out)
            if message['type'] == 'executing':
                data = message['data']
                if data['prompt_id'] == prompt_id:
                    if data['node'] is None:
                        break #Execution is done
                    else:
                        current_node = data['node']
        else:
            if current_node == OUTPUT_NODE_WORKFLOW_ID:
                images_output = output_images.get(current_node, [])
                images_output.append(out[8:])
                output_images[current_node] = images_output

    return output_images


def save_images(images, output_folder):
    for _, image in images.items():
        with open(os.path.join(output_folder, f"{uuid.uuid4()}.png"), "wb") as f:
            f.write(image[0])

def load_prompt(workflow_name):
    with open(os.path.join(ASSETS_FOLDER, 'workflows', workflow_name), 'r') as f:
        prompt = json.load(f)
    # set the text prompt for our positive CLIPTextEncode
    prompt[INPUT_NODE_WORKFLOW_ID]["inputs"]["text"] = content_prompt + " " + style_prompt

    # set the seed for our KSampler node
    prompt[SEED_NODE_WORKFLOW_ID]["inputs"]["seed"] = seed

    # set the steps for our KSampler node
    prompt[STEPS_NODE_WORKFLOW_ID]["inputs"]["steps"] = steps

    original_width = prompt[IMAGE_SIZE_NODE_WORKFLOW_ID]["inputs"]["width"]
    original_height = prompt[IMAGE_SIZE_NODE_WORKFLOW_ID]["inputs"]["height"]

    if ratio == ImageRatio.LANDSCAPE:
        width = int(original_width)
        height = int(original_height * (9/16))
    elif ratio == ImageRatio.PORTRAIT:
        height = int(original_height)
        width = int(original_width * (16/9))
    else:
        width = int(original_width)
        height = int(original_height)

    prompt[IMAGE_SIZE_NODE_WORKFLOW_ID]["inputs"]["width"] = width
    prompt[IMAGE_SIZE_NODE_WORKFLOW_ID]["inputs"]["height"] = height

    return prompt


prompt = load_prompt(workflow_name)
ws = websocket.WebSocket()
ws.connect("ws://{}/ws?clientId={}".format(SERVER_ADDRESS, client_id))
images = get_images(ws, prompt)
ws.close() # for in case this example is used in an environment where it will be repeatedly called, like in a Gradio app. otherwise, you'll randomly receive connection timeouts
#Commented out code to display the output images:
save_images(images, os.path.join(ASSETS_FOLDER, 'images'))
