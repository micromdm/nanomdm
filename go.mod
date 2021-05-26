module github.com/jessepeterson/nanomdm

go 1.15

require (
	github.com/RobotsAndPencils/buford v0.14.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/groob/plist v0.0.0-20210519001750-9f754062e6d6
	go.mozilla.org/pkcs7 v0.0.0-20200128120323-432b2356ecb1
)

replace go.mozilla.org/pkcs7 v0.0.0-20200128120323-432b2356ecb1 => github.com/omorsi/pkcs7 v0.0.0-20210217142924-a7b80a2a8568
