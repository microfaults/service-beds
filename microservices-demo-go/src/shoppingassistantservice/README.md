# Shopping Assistant Service

## Setup
Install gcloud and authenticate. Liam will need to add you as a member to the project, so you're allowed to make the AI API calls. The following command will do that for you.

```
make setup
```

This will:
1. Install the Google Cloud SDK via Homebrew (if not already installed)
2. Log you into gcloud
3. Set up Application Default Credentials (ADC) used by the Go SDK

Then copy the env file and fill in the project ID:

```
cp .env.example .env
# Edit .env and set PROJECT_ID to the value Liam gives you
```

Then register the quota project:

```
make set-quota-project
```

## Running

```
make docker-run
```

Builds the image with the correct google cloud credentials.

## Testing

Send a request without an image.

```
curl -X POST http://localhost:8082/ \
  -H "Content-Type: application/json" \
  -d '{"message": "I would like to buy some plants for my room"}'
```

Send a request with a room image:

```
curl -X POST http://localhost:8082/ \
  -H "Content-Type: application/json" \
  -d '{"message": "I would like to buy some plants for my room", "image": "https://hips.hearstapps.com/hmg-prod/images/pamela-forman-living-room-694963058722c.jpg?crop=0.669xw:1.00xh;0.109xw,0&resize=640:*"}'
```

The response will contain a recommendation and a list of product IDs in the format `[ID1], [ID2], [ID3]`.
