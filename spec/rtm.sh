# create new room and return its id
curl -X POST http://localhost:8087/api/rtaskman/room \
-H "X-User: alice"
# -> e42fd95d-4bf2-4c31-b05c-102795dd5772

ROOM_ID='e42fd95d-4bf2-4c31-b05c-102795dd5772'


# list all active rooms
curl -X GET http://localhost:8087/api/rtaskman/room


# delete a room by setting deleted_at and deleted_by
curl -X DELETE http://localhost:8087/api/rtaskman/room/${ROOM_ID} \
-H "X-User: alice"





# create new series with all mandatory fields and return its id
curl -X POST http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series \
-H "X-User: jim" \
-H "Content-Type: application/json" \
-d '{
  "name": "Series 2"
}'


# create new series with all fields and return its id
curl -X POST http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series \
-H "X-User: bob" \
-H "Content-Type: application/json" \
-d '{
  "name": "Series 3",
  "description": "This is a description for Series 3",
  "color": "#ff0000",
  "tag_list": ["tag3"],
  "target_interval": "P3DT8H",
  "meta": {"key": "value"}
}'


# list all active series in a specific room
curl -X GET http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series


# get a specific series by id
curl -X GET http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID}


# delete a series by setting deleted_at and deleted_by
curl -X DELETE http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID} \
-H "X-User: bob"


# update series with all fields
curl -X PUT http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID} \
-H "Content-Type: application/json" \
-d '{
  "name": "Updated Series Name updated",
  "description": "Updated description",
  "color": "#00ff00",
  "tag_list": ["tag1", "tag2"],
  "target_interval": "P1DT12H",
  "meta": {"updated_key": "updated_value"}
}'


# update series with mandatory fields only
curl -X PUT http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID} \
-H "Content-Type: application/json" \
-d '{
  "name": "new name"
}'





# create new event with all mandatory fields and return the created event
curl -X POST http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID}/event \
-H "X-User: eve" \
-d '{}'


# create new event with all optional fields and return the created event
curl -X POST http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID}/event \
-H "X-User: eve" \
-d '{
  "created_at": "2023-10-01T12:00:00Z",
  "meta": {"event_key": "event_value"}
}'


# list all events for a specific series
curl -X GET http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID}/event


# delete an event
curl -X DELETE http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID}/event/${EVENT_ID}


# update an event
curl -X PUT http://localhost:8087/api/rtaskman/room/${ROOM_ID}/series/${SERIES_ID}/event/${EVENT_ID} \
-d '{
  "created_at": "2023-10-01T12:00:00+01:00",
  "meta": {"event_key": "event_value"}
}'



# get an iCalendar feed for specific series
curl -X GET http://localhost:8087/api/rtaskman/room/${ROOM_ID}/ical?series_id=${SERIES_ID1}&series_id=${SERIES_ID2}
