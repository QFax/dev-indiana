# Indiana

A transparent Gemini proxy that round-robins keys

## Features

- **Transparent Proxy**: Forwards requests to the Gemini API and is compatible with official clients.
- **Round-Robin API Keys**: Distributes requests across a pool of Gemini API keys.
- **Delayed Rate Limiting**: Delays requests to avoid exceeding rate limits, rather than dropping them.
- **Usage Statistics**: Tracks per-minute and daily usage statistics for each Gemini API key.
- **Dockerized**: Easy to deploy and manage with Docker and Docker Compose.

## Tech Stack

- **Application**: Go
- **Caching and Data Store**: Valkey
- **Containerization**: Docker, Docker Compose

## Getting Started

### Prerequisites

- Docker
- Docker Compose
- Go (for local development)

### Setup

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/your-username/gemini-api-proxy.git
    cd gemini-api-proxy
    ```

2.  **Create a `.env` file**:
    - Create a `.env` file in the root of the project, or copy it from .env.example.
    - Populate it with the necessary values, as described in `ENV_SCHEMA.md`.

3.  **Build and run the application**:
    ```bash
    docker-compose up --build
    ```
    The application will be available at `http://localhost:8080`.

## Nginx Proxy Configuration

For production deployments, it is recommended to use a reverse proxy like Nginx to handle SSL termination and proxy requests to the application.

Here is an example Nginx server block:

```nginx
server {
    listen 443 ssl;
    server_name your_domain.com;

    ssl_certificate /path/to/your/fullchain.pem;
    ssl_certificate_key /path/to/your/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## API Usage

To use the proxy, send your requests to the proxy's URL instead of the Gemini API URL. You must include your API key in the `x-goog-api-key` header.