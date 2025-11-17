# This probably wont work right now, in the middle of testing.

# Build
first, clone the repo:
```
git clone https://github.com/FloofyIV/tracker
cd tracker
```
then, build it with:
```
go build -ldflags="-s -w" .
```

# Usage
set the webhook: ```WEBHOOK="https://discord.com/api/webhooks/xxx/xxx"```<br />
set the placeid: ```PLACE="155615604"```<br />
### or: (recommended)<br />
set the webhook: ```export WEBHOOK="https://discord.com/api/webhooks/xxx/xxx"```<br />
set the placeid: ```export PLACE="155615604"```<br />

## Example:
```PLACE="155615604" WEBHOOK="https://discord.com/api/webhooks/xxx/xxx" ./tracker```<br />
### or: (recommended)<br />
```
$ export PLACE="155615604"
$ explort WEBHOOK="https://discord.com/api/webhooks/xxx/xxx"
$ ./tracker
```
