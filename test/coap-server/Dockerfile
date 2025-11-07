FROM python:3.11-slim

# Install aiocoap for CoAP server
RUN pip install --no-cache-dir aiocoap

# Create server script
COPY server.py /app/server.py

WORKDIR /app

EXPOSE 5683/udp

CMD ["python", "server.py"]
