# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Establecer directorio de trabajo
WORKDIR /app

# Copiar go.mod y go.sum primero para cachear dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar todo el código fuente
COPY . .

# Compilar el binario
RUN go build -o search-api .

# Stage 2: Runtime
FROM alpine:latest

# Instalar ca-certificates para HTTPS requests
RUN apk --no-cache add ca-certificates

# Establecer directorio de trabajo
WORKDIR /app

# Copiar el binario desde el stage de build
COPY --from=builder /app/search-api .

# Exponer el puerto 8082
EXPOSE 8082

# Ejecutar el binario
CMD ["./search-api"]

