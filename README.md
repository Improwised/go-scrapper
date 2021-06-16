# go-scrapper
Scrapy project with go lang

# Setup

Install modules
```
go mod vendor
```

# Run

To run spider
```
go run <spiderfile>
```

Example:
```
go run main.go -s USER_AGENT='Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/32.0.1700.72 Safari/537.36' -a place='eyJwcm9maWxlX2tleSI6Imh0dHBzOlwvXC93d3cueWVscC5jb21cL2JpelwvYWxsLWFib3V0LWV5ZXMtbWFzb24tY2l0eS1tYXNvbi1jaXR5IiwibGFzdF9yZXZpZXdfaGFzaGVzIjpbXSwicGVyc29uYSI6eyJ1c2VybmFtZSI6bnVsbCwicGFzc3dvcmQiOm51bGwsImNvb2tpZXMiOltdLCJhZGRpdGlvbmFsX2Nvb2tpZXMiOltdLCJwcm94eSI6ImJhMThmMmRiZTNmODQwMjU5YTY3NWU2OWYyMGI0NDFhOkBvZG1hcmtqLmNyYXdsZXJhLmNvbTo4MDEwIiwib3RoZXJfcHJveGllcyI6WyIyYzAyZmE2ZWIzYjg0YzdhOTUzMDk0MmY0Mzk1ZDVjODpAb2RtYXJrai5jcmF3bGVyYS5jb206ODAxMCJdfSwiYnVzaW5lc3NfaWQiOjEsImJ1c2luZXNzX25hbWUiOiJBbGwgQWJvdXQgRXllcyAtIE1hc29uIENpdHkiLCJhZGRyZXNzIjp7ImNpdHkiOiJNYXNvbiBDaXR5Iiwic3RhdGUiOiJJQSIsInN0cmVldCI6IjIyNjEgNHRoIFN0IFNXIiwiemlwIjoiNTA0MDEifSwidHlwZSI6ImFwaXxwdWJsaWN8cHJpdmF0ZSIsImNsaWVudF9pZCI6NCwiYmF0Y2hfaWQiOjYsInRhc2tfaWQiOjYsImZpcnN0X3BhZ2Vfb25seSI6MCwidGltZW91dCI6NDAwMH0=' -o "somthing.jl"

```
#### Install Flask
Development environment also needs flask app to run locally. So open new tab in terminal and install, run flask.

* Install Flask
  `pip install flask`
* Run Flask locally:
  `FLASK_APP=match_service_stub.py flask run --host=127.0.0.1 --port=9999`

  If Development environment,set MATCH_SERVICE_URL in .env :
  	`MATCH_SERVICE_URL=http://127.0.0.1:9999/match`