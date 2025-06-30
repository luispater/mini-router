# Mini Router - AI Model Router

## Overview

Mini Router is a lightweight, high-performance AI model router built with Go. It is designed to provide a unified API endpoint for various large language models (LLMs), offering OpenAI API compatibility. This allows clients to interact with different AI providers through a single, consistent interface. The router supports features like model load balancing, rate limiting, and dynamic configuration via a YAML file.

## Features

*   **OpenAI API Compatibility**: Exposes a standard OpenAI-compatible endpoint (`/v1/chat/completions`), allowing seamless integration with existing tools and libraries that support the OpenAI API.
*   **Multi-Model Support**: Configure and manage multiple AI models from different providers within a single instance.
*   **Load Balancing**: Implements round-robin load balancing for models that have multiple provider API keys or configurations, enhancing reliability and distributing the load.
*   **Dynamic Configuration**: All settings, including server configuration, models, and API keys, are managed through a single `config.yaml` file, which is loaded at startup.
*   **Streaming and Non-Streaming Support**: Handles both streaming (`text/event-stream`) and standard JSON responses for chat completions.
*   **Health Check**: A `/health` endpoint to monitor the status of the router.
*   **Authentication**: Secures the chat completion endpoint with API key authentication.
*   **Extensible Provider System**: Designed to be easily extensible with new AI providers.

## Getting Started

### Prerequisites

*   [Go](https://golang.org/dl/) (version 1.24 or later is recommended)

### Installation & Running

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/luispater/mini-router.git
    cd mini-router
    ```

2.  **Configure the application:**
    Create and edit the `config.yaml` file based on the instructions in the **Configuration** section below.

3.  **Build the application:**
    ```bash
    go build -o mini-router
    ```

4.  **Run the application:**
    ```bash
    ./mini-router
    ```
    The server will start on the port specified in your `config.yaml`.

## Configuration (`config.yaml`)

The application is configured using the `config.yaml` file. Here is a detailed breakdown of the configuration options.

### `server`

This section configures the HTTP server settings.

| Parameter         | Type     | Description                                            | Example |
| ----------------- | -------- | ------------------------------------------------------ | ------- |
| `port`            | `string` | The port the server will listen on.                    | `"8316"`  |
| `shutdown_timeout`| `string` | The graceful shutdown timeout (e.g., `10s`, `1m`).     | `10s`   |

**Example:**
```yaml
server:
  port: "8316"
  shutdown_timeout: 10s
```

### `models`

This is a list of AI models that the router will manage.

| Parameter                 | Type        | Description                                                                                             |
| ------------------------- | ----------- | ------------------------------------------------------------------------------------------------------- |
| `id`                      | `integer`   | A unique identifier for the model configuration entry.                                                  |
| `name`                    | `string`    | The model name used in the API request (e.g., `gemini-2.5-pro:free`).                                   |
| `provider_model_name`     | `string`    | The actual model name that the provider expects (e.g., `gemini-2.5-pro`).                               |
| `display_name`            | `string`    | A user-friendly name for display purposes.                                                              |
| `description`             | `string`    | A brief description of the model.                                                                       |
| `supports_chat`           | `boolean`   | Whether the model supports chat completions.                                                            |
| `supports_completion`     | `boolean`   | Whether the model supports standard text completions.                                                   |
| `supports_input_image`    | `boolean`   | Whether the model supports image inputs.                                                                |
| `support_google_thinking` | `boolean`   | Enables special handling for Google's "thinking" or reasoning features.                                 |
| `rpm`                     | `integer`   | Requests Per Minute limit for this model.                                                               |
| `tpm`                     | `integer`   | Tokens Per Minute limit for this model.                                                                 |
| `rpd`                     | `integer`   | Requests Per Day limit for this model.                                                                  |
| `input_price_per_token`   | `float`     | The cost per input token.                                                                               |
| `output_price_per_token`  | `float`     | The cost per output token.                                                                              |
| `max_tokens`              | `integer`   | The maximum number of tokens the model can generate in a single response.                               |
| `context_length`          | `integer`   | The maximum context length (in tokens) the model supports.                                              |
| `supported_parameters`    | `[]string`  | A list of API parameters supported by this model (e.g., `tools`, `temperature`).                        |
| `provider_api_key`        | `[]string`  | A list of API keys for the backend provider. The router will use these in a round-robin fashion.        |
| `is_openai_compatibility` | `boolean`   | Set to `true` if the provider's API is OpenAI-compatible.                                               |
| `base_url`                | `string`    | The base URL of the provider's API endpoint.                                                            |
| `enabled`                 | `boolean`   | If `true`, this model configuration is active and can be used.                                          |
| `visible`                 | `boolean`   | If `true`, this model will be listed in the `/v1/models` endpoint.                                      |

**Example:**
```yaml
models:
  - id: 4
    name: "gemini-2.5-pro:free"
    provider_model_name: "gemini-2.5-pro"
    display_name: "Gemini 2.5 Pro(Free)"
    description: "Google's Gemini 2.5 Pro"
    supports_chat: true
    supports_completion: true
    supports_input_image: true
    support_google_thinking: true
    rpm: 5
    tpm: 250000
    rpd: 100
    input_price_per_token: 0.0
    output_price_per_token: 0.0
    max_tokens: 65536
    context_length: 1048576
    supported_parameters: ["tools", "tool_choice", "max_tokens", "temperature", "top_p", "stop", "frequency_penalty", "presence_penalty", "seed", "response_format", "structured_outputs"]
    provider_api_key:
      - "AIzaSy...S6cBqxM4"
      - "AIzaSy...J7amzb2w"
    is_openai_compatibility: true
    base_url: "https://generativelanguage.googleapis.com/v1beta/openai"
    enabled: true
    visible: true
```

### `api_keys`

This section defines the API keys that clients will use to authenticate with the Mini Router itself.

| Parameter | Type      | Description                                                              |
| --------- | --------- | ------------------------------------------------------------------------ |
| `id`      | `integer` | A unique identifier for the API key.                                     |
| `key`     | `string`  | The API key string. Should be prefixed (e.g., `sk-or-v1-...`).           |
| `name`    | `string`  | A descriptive name for the key.                                          |
| `is_active`| `boolean` | If `true`, the key is active and can be used for authentication.         |
| `user_id` | `integer` | An associated user ID.                                                   |
| `rps`, `rpm`, `rph`, `rpd` | `integer` | Rate limits for this key (requests per second/minute/hour/day). `0` means no limit. |
| `tps`, `tpm`, `tph`, `tpd` | `integer` | Token limits for this key (tokens per second/minute/hour/day). `0` means no limit. |

**Example:**
```yaml
api_keys:
  - id: 1
    key: "sk-or-v1-..."
    name: "Default"
    is_active: true
    user_id: 1
    rpm: 0 # No limit
```

## API Endpoints

### Health Check

*   **Endpoint**: `GET /health`
*   **Description**: Returns the operational status of the server.
*   **Authentication**: None
*   **Success Response (200 OK)**:
    ```json
    {
      "status": "ok",
      "time": "2025-07-01T16:05:29+08:00"
    }
    ```

### List Models

*   **Endpoint**: `GET /v1/models`
*   **Description**: Retrieves a list of all `visible` and `enabled` models from the configuration.
*   **Authentication**: None
*   **Success Response (200 OK)**:
    ```json
    {
      "object": "list",
      "data": [
        {
          "id": "gemini-2.5-pro:free",
          "object": "model",
          "name": "Gemini 2.5 Pro(Free)",
          "description": "Google's Gemini 2.5 Pro",
          "supported_parameters": [
            "tools",
            "tool_choice",
            "max_tokens",
            // ... other parameters
          ],
          "context_length": 1048576,
          "max_completion_tokens": 65536,
          // ... other model details
        }
      ]
    }
    ```

### Chat Completions

*   **Endpoint**: `POST /v1/chat/completions`
*   **Description**: Creates a model response for the given chat conversation. This endpoint is compatible with the OpenAI Chat Completions API.
*   **Authentication**: Required. Provide an API key from the `api_keys` configuration in the `Authorization` header as a Bearer token.
    ```
    Authorization: Bearer sk-or-v1-...
    ```
*   **Request Body**: Standard OpenAI chat completion request body.
    ```json
    {
      "model": "gemini-2.5-pro:free",
      "messages": [
        {
          "role": "user",
          "content": "Hello!"
        }
      ],
      "stream": false
    }
    ```
*   **Success Response**: Standard OpenAI chat completion response (or a `text/event-stream` if `stream: true`).

## Dependencies

This project relies on several open-source libraries, including:

*   [Gin](https://github.com/gin-gonic/gin): A high-performance HTTP web framework.
*   [gjson](https://github.com/tidwall/gjson): A fast JSON parser.
*   [sjson](https://github.com/tidwall/sjson): A library to set JSON values.
*   [gojsonschema](https://github.com/xeipuuv/gojsonschema): A JSON Schema validator.
*   [yaml.v3](https://gopkg.in/yaml.v3): A YAML support library for Go.

## License

This project is licensed under the MIT License. Refer to the [LICENSE](LICENSE) file for details.
