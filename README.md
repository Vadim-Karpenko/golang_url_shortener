# Golang URL Shortener

An implementation of URL Shortener in Golang, Gin, and Redis. It allows you to shorten long URLs into shorter, more manageable ones and set access limits.

## Features

- Generate short URLs for long URLs
- Set maximum access limits for URLs
- Set maximum access per hour limits
- Set expiration time for URLs

## Prerequisites

- [Go](https://golang.org/dl/) (version 1.16 or higher)
- [Redis](https://redis.io/download) (version 6 or higher)
- [Gin](https://github.com/gin-gonic/gin) (version 1.7 or higher)

## Installation

1. **Clone the repository**:
    ```sh
    git clone https://github.com/yourusername/url-shortener.git
    cd url-shortener
    ```

2. **Install dependencies**:
    ```sh
    go mod tidy
    ```

3. **Run Redis server**:
    ```sh
    redis-server
    ```

4. **Run the application**:
    ```sh
    go run main.go
    ```

## Usage

### Create a Short URL

- **Endpoint**: `POST /create`
- **Parameters**:
  - `long_url` (required): The original long URL.
  - `max_access` (optional): Maximum number of times the short URL can be accessed. Default: -1.
  - `max_per_hour` (optional): Maximum number of times the short URL can be accessed per hour. Default: -1.
  - `max_age` (optional): Maximum age of the short URL in seconds. Default: 3600.

- **Example**:
    ```sh
    curl -X POST http://localhost:8080/create \
    -d "long_url=https://example.com" \
    -d "max_access=10" \
    -d "max_per_hour=5" \
    -d "max_age=3600"
    ```

- **Response**:
    ```json
    {"token": "BANVmpyh"}
    ```

### Use Short URL

- **Endpoint**: `GET /:token`
- **Parameters**:
  - `token` (required): The token you got at the creation step.

- **Example**:
    ```sh
    curl -X GET http://localhost:8080/BANVmpyh
    ```

## Testing

To run the tests for this URL shortener application, follow these steps:

1. **Navigate to the project directory**:
    ```sh
    cd ./golang_url_shortener
    ```

2. **Run the tests**:
    ```sh
    go test
    ```



## Configuration

The following configuration options are available in the `main.go` file:

- `redisAddr`: Address of the Redis server (default: `localhost:6379`)
- `redisPassword`: Password for the Redis server (default: `""`)
- `redisDB`: Redis database number (default: `0`)

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

1. **Fork the repository**.
2. **Create a new branch**:
    ```sh
    git checkout -b feature/your-feature-name
    ```
3. **Commit your changes**:
    ```sh
    git commit -m "Add some feature"
    ```
4. **Push to the branch**:
    ```sh
    git push origin feature/your-feature-name
    ```
5. **Open a pull request**.
