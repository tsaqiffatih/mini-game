1. room/create:
    curl -X POST http://localhost:8080/room/create -H "Content-Type: application/json" -d '{
    "game_type": "chess",
    "player_id": "player1"
    }'

2. room/join:
    curl -X POST http://localhost:8080/room/join -H "Content-Type: application/json" -d '{
    "room_id": "B6QX48A",
    "player_id": "player2"
    }'

3. create/user:
    curl -X POST http://localhost:8080/create/user -H "Content-Type: application/json" -d '{
    "player_id": "player1"
    }'

4. websocket:
    => wscat -c "ws://localhost:8080/ws?room_id=L7PVSR0&player_id=tsaqif"

    => {"action": "CHAT_MESSAGE","message": "Hello, everyone!","sender": {"id":"player1"}}

    => tictactoe move:
        {"action": "TICTACTOE_MOVE","message": {"room_id": "room1","player_id": "player1","row": 1,"col": 1},"sender": {"id": "player1"}}