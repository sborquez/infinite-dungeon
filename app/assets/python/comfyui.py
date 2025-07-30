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

# Node types for the new workflow format
OUTPUT_NODE_WORKFLOW_TYPE = "SaveImageWebsocket"

class ImageRatio(StrEnum):
    SQUARE = "SQUARE"
    LANDSCAPE = "LANDSCAPE"
    PORTRAIT = "PORTRAIT"

client_id = str(uuid.uuid4())
content_prompt = "A beautiful space station in the sky, seen from the ground"
workflow_name = "default.json"
seed = random.randint(0, 1000000)
steps = 15
ratio = str(ImageRatio.LANDSCAPE)
size = 512


def queue_prompt(prompt):
    p = {"prompt": prompt, "client_id": client_id}
    data = json.dumps(p).encode('utf-8')
    req =  urllib.request.Request("http://{}/prompt".format(SERVER_ADDRESS), data=data)
    return json.loads(urllib.request.urlopen(req).read())

def find_output_node_id(prompt):
    """Find the node ID of the SaveImageWebsocket node"""
    for node_id, node_data in prompt.items():
        if isinstance(node_data, dict) and "class_type" in node_data:
            if node_data["class_type"] == OUTPUT_NODE_WORKFLOW_TYPE:
                return node_id
    return None

def get_images(ws, prompt):
    prompt_id = queue_prompt(prompt)['prompt_id']
    output_node_id = find_output_node_id(prompt)
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
            if current_node == output_node_id:
                if current_node not in output_images:
                    output_images[current_node] = []
                output_images[current_node].append(out[8:])

    return output_images


def save_images(images, output_folder):
    for _, image in images.items():
        with open(os.path.join(output_folder, f"{uuid.uuid4()}.png"), "wb") as f:
            f.write(image[0])

def find_node_by_title(prompt, title):
    """Find a node by its _meta.title field"""
    for node_id, node_data in prompt.items():
        if isinstance(node_data, dict) and "_meta" in node_data:
            if "title" in node_data["_meta"] and node_data["_meta"]["title"] == title:
                return node_id, node_data
    return None, None

def update_node_value(prompt, title, value):
    """Update a node's input value by finding it via _meta.title"""
    node_id, node_data = find_node_by_title(prompt, title)
    if node_data and "inputs" in node_data:
        node_data["inputs"]["value"] = value
        print(f"Updated node '{title}' (ID: {node_id}) with value: {value}")
        return True
    else:
        print(f"Warning: Could not find node with title '{title}'")
        return False

def update_ksampler_steps(prompt, steps_value):
    """Update the KSampler node's steps value"""
    for node_id, node_data in prompt.items():
        if isinstance(node_data, dict) and "class_type" in node_data:
            if node_data["class_type"] == "KSampler" and "inputs" in node_data:
                node_data["inputs"]["steps"] = steps_value
                print(f"Updated KSampler node (ID: {node_id}) steps to: {steps_value}")
                return True
    print("Warning: Could not find KSampler node")
    return False

def load_prompt(workflow_name):
    with open(os.path.join(ASSETS_FOLDER, 'workflows', workflow_name), 'r') as f:
        prompt = json.load(f)

    # Update individual nodes by their _meta.title
    update_node_value(prompt, "Ratio", ratio)
    update_node_value(prompt, "ContentPrompt", content_prompt)
    update_node_value(prompt, "Seed", seed)
    update_node_value(prompt, "Size", float(size))

    # Update steps in the KSampler node
    update_ksampler_steps(prompt, steps)

    return prompt


prompt = load_prompt(workflow_name)
ws = websocket.WebSocket()
ws.connect("ws://{}/ws?clientId={}".format(SERVER_ADDRESS, client_id))
images = get_images(ws, prompt)
ws.close() # for in case this example is used in an environment where it will be repeatedly called, like in a Gradio app. otherwise, you'll randomly receive connection timeouts
#Commented out code to display the output images:
save_images(images, os.path.join(ASSETS_FOLDER, 'images'))
