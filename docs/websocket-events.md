# Real-time Events with WebSocket

This document describes the WebSocket implementation for real-time events in the Stories Service.

## Overview

The service now supports real-time notifications for:
- **story.viewed**: When someone views your story
- **story.reacted**: When someone reacts to your story

## WebSocket Connection

### Endpoint
```
ws://localhost:8080/ws?token=YOUR_JWT_TOKEN
```

### Authentication
- WebSocket connections require JWT authentication via query parameter
- Use the same JWT token from login/signup endpoints
- Connection will be rejected if token is invalid or missing

### Example JavaScript Client
```javascript
// Get JWT token from login/signup response
const token = "your_jwt_token_here";

// Connect to WebSocket
const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

ws.onopen = function() {
    console.log('Connected to WebSocket');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Real-time event received:', data);
    
    switch(data.type) {
        case 'story.viewed':
            console.log(`User ${data.data.viewer_id} viewed your story ${data.data.story_id}`);
            break;
        case 'story.reacted':
            console.log(`User ${data.data.user_id} reacted with ${data.data.emoji} to your story ${data.data.story_id}`);
            break;
    }
};

ws.onclose = function() {
    console.log('WebSocket connection closed');
};

ws.onerror = function(error) {
    console.error('WebSocket error:', error);
};
```

## Event Types

### story.viewed
Sent to story author when someone views their story.

```json
{
    "type": "story.viewed",
    "data": {
        "story_id": "550e8400-e29b-41d4-a716-446655440000",
        "viewer_id": "user123",
        "viewed_at": "2023-10-01T12:00:00Z"
    },
    "timestamp": "2023-10-01T12:00:00Z"
}
```

### story.reacted
Sent to story author when someone reacts to their story.

```json
{
    "type": "story.reacted",
    "data": {
        "story_id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "user123",
        "emoji": "❤️",
        "reacted_at": "2023-10-01T12:00:00Z"
    },
    "timestamp": "2023-10-01T12:00:00Z"
}
```

## Usage Flow

1. **Connect**: Establish WebSocket connection with JWT token
2. **Listen**: Listen for incoming real-time events
3. **Act**: Handle events in your client application (show notifications, update UI, etc.)

## Important Notes

- Only story authors receive notifications for their stories
- Self-actions (viewing/reacting to your own story) don't trigger notifications
- Events are only sent to currently connected users
- Connection is automatically managed (ping/pong, reconnection handling)
- One connection per user (new connection replaces old one)

## Testing the WebSocket

### Using curl for HTTP endpoints
```bash
# First, get a JWT token
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Create a story
curl -X POST http://localhost:8080/stories \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello World","visibility":"PUBLIC","audience_user_ids":[]}'

# View a story (triggers real-time event)
curl -X POST http://localhost:8080/stories/STORY_ID/view \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# React to a story (triggers real-time event)
curl -X POST http://localhost:8080/stories/STORY_ID/reactions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"emoji":"❤️"}'
```

### WebSocket Test Page

Save this as `websocket-test.html` and open in a browser:

```html
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Test</title>
</head>
<body>
    <h1>Stories Service WebSocket Test</h1>
    
    <div>
        <label>JWT Token:</label><br>
        <input type="text" id="token" placeholder="Paste your JWT token here" style="width: 500px;">
        <button onclick="connect()">Connect</button>
        <button onclick="disconnect()">Disconnect</button>
    </div>
    
    <div style="margin-top: 20px;">
        <strong>Connection Status:</strong> <span id="status">Disconnected</span>
    </div>
    
    <div style="margin-top: 20px;">
        <h3>Events Log:</h3>
        <div id="events" style="border: 1px solid #ccc; padding: 10px; height: 300px; overflow-y: scroll; background: #f9f9f9;"></div>
    </div>

    <script>
        let ws = null;
        const statusEl = document.getElementById('status');
        const eventsEl = document.getElementById('events');
        
        function connect() {
            const token = document.getElementById('token').value;
            if (!token) {
                alert('Please enter a JWT token');
                return;
            }
            
            if (ws) {
                ws.close();
            }
            
            ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);
            
            ws.onopen = function() {
                statusEl.textContent = 'Connected';
                statusEl.style.color = 'green';
                addEvent('Connected to WebSocket', 'system');
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                addEvent(JSON.stringify(data, null, 2), 'event');
            };
            
            ws.onclose = function() {
                statusEl.textContent = 'Disconnected';
                statusEl.style.color = 'red';
                addEvent('WebSocket connection closed', 'system');
            };
            
            ws.onerror = function(error) {
                statusEl.textContent = 'Error';
                statusEl.style.color = 'red';
                addEvent('WebSocket error: ' + error, 'error');
            };
        }
        
        function disconnect() {
            if (ws) {
                ws.close();
                ws = null;
            }
        }
        
        function addEvent(message, type) {
            const time = new Date().toLocaleTimeString();
            const div = document.createElement('div');
            div.style.marginBottom = '5px';
            div.style.padding = '5px';
            
            if (type === 'system') {
                div.style.backgroundColor = '#e7f3ff';
                div.style.color = '#0066cc';
            } else if (type === 'error') {
                div.style.backgroundColor = '#ffe7e7';
                div.style.color = '#cc0000';
            } else {
                div.style.backgroundColor = '#e7ffe7';
                div.style.color = '#006600';
                div.style.fontFamily = 'monospace';
            }
            
            div.innerHTML = `<strong>${time}:</strong> <pre style="margin: 0; white-space: pre-wrap;">${message}</pre>`;
            eventsEl.appendChild(div);
            eventsEl.scrollTop = eventsEl.scrollHeight;
        }
    </script>
</body>
</html>
```

## Architecture

### Components
- **WebSocket Hub**: Manages all active connections
- **Event Publisher**: Publishes events to connected users
- **WebSocket Handler**: Handles connection upgrades and authentication
- **Enhanced Story Handlers**: Emit events after database operations

### Flow
```
User Action → Handler → Database Update → Event Publisher → Hub → WebSocket → Client
```

This implementation provides real-time, scalable event delivery while maintaining security and performance.
