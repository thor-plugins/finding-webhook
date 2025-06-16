# Finding Webhook Plugin

## Description

This plugin uploads all findings (including content that was matched on) to a webhook URL.

## Usage

Set the server URL using the `THOR_PLUGIN_FINDING_WEBHOOK_URL` environment variable.
The plugin will send a POST request to this URL with the findings data.
The request will contain a multipart/form-data body with the following fields:
- `finding`: The finding that was reported, in JSON format. This will be a marshaled `thorlog.Finding` object.
- `content`: The content of the object that was matched on. If no content exists, this will not be present.

An example server that receives the findings can be found in the `server` directory.
