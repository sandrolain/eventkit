#!/usr/bin/env python3
"""
Simple CoAP test server for eventkit testing.
Responds to POST requests with 2.05 Content and echoes back the payload.
"""

import asyncio
import aiocoap.resource as resource
import aiocoap


class EchoResource(resource.Resource):
    """Resource that echoes back the payload"""
    
    async def render_post(self, request):
        """Handle POST requests"""
        print(f"Received POST: {request.payload.decode('utf-8', errors='ignore')}")
        
        response_payload = b"OK"
        if request.payload:
            response_payload = request.payload
            
        return aiocoap.Message(code=aiocoap.CONTENT, payload=response_payload)
    
    async def render_get(self, request):
        """Handle GET requests"""
        return aiocoap.Message(code=aiocoap.CONTENT, payload=b"CoAP server ready")


async def main():
    """Start the CoAP server"""
    root = resource.Site()
    root.add_resource(['.well-known', 'core'], resource.WKCResource(root.get_resources_as_linkheader))
    root.add_resource(['test'], EchoResource())
    root.add_resource(['echo'], EchoResource())
    
    await aiocoap.Context.create_server_context(root, bind=('0.0.0.0', 5683))
    
    print("CoAP server started on 0.0.0.0:5683")
    print("Available resources: /test, /echo")
    
    # Run forever
    await asyncio.get_running_loop().create_future()


if __name__ == "__main__":
    asyncio.run(main())
