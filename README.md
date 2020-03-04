# sparty

Spotify Party! Let your friends add Spotify songs to your queue from their own devices.

## How it works

This project exposes a very basic API with just a single endpoint: `POST /enqueue`:

```bash
curl -H "Authorization: Token <token>" -X "POST" "http://localhost:8080/enqueue?url=spotify:track:1301WleyT98MSxVHPZCA6M
``` 

The `url` can be obtained from Spotify, for example by performing a [search](https://developer.spotify.com/documentation/web-api/reference/search/search/).

Once `spartyd` receives this request, it does some very basic valiation and responds with a `204 No Content`. The endpoint does NOT make a request to the Spotify Web API directly: it only accepts it for delivery. A job is created, and a worker will pick this up to actually send it over to Spotify.    

## Requirements

* Go 1.13
* Spotify premium account

## Config

Simply build the daemon in `cmd/spartyd` and run it with these environment variables set:

* `PORT` (optional, defaults to 8080: port to listen on for API requests)
* `SPARTY_AUTH_TOKEN` (arbitrary token to authenticate with API by passing it in a header `Authorization: Token <token>`)
* `SPOTIFY_CLIENT_ID`
* `SPOTIFY_CLIENT_SECRET`
* `SPOTIFY_REFRESH_TOKEN`

See [this guide](https://developer.spotify.com/documentation/general/guides/authorization-guide/) by Spotify to learn how to obtain these `SPOTIFY_` values.

## Final note

The Spotify Web API [endpoint](https://developer.spotify.com/documentation/web-api/reference/player/add-to-queue/) used to enqueue songs is still in beta, and is thus subject to change by Spotify without prior notice. This means this application may stop working at any moment. If it does and you'd like to fix it, the endpoint's documentation is the first place to look. 
