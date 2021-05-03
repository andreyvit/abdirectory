server=h3

ship: build-linux
	scp /tmp/abdirectory-linux-amd64 '$(server):~/'
	ssh $(server) 'bash -s --' < _deploy/abdirectory.sh

reconf:
	ssh $(server) 'bash -s --' < _deploy/abdirectory.sh

build-linux:
	GOOS=linux GOARCH=amd64 go build -o /tmp/abdirectory-linux-amd64 ./cmd/abdirectory-update

logs:
	ssh $(server) 'sudo journalctl -f -u lmtchat'

restart-caddy:
	ssh $(server) 'sudo systemctl restart caddy'

upload-secret:
	ssh $(server) 'sudo install -m600 -oandreyvit -gandreyvit /dev/stdin /srv/abdirectory/secrets/abdirectory_secret.json' < client_secret.json
