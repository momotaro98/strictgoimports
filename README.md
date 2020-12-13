# strictgoimports

## Install

```
$ go get -u github.com/momotaro98/strictgoimports/cmd/strictgoimports
```

## Usage

### Use as checker

```shell script
$ strictgoimports -exclude "*_mock.go,*.pb.go" -exclude-dir "testmock" -local "github.com/momotaro98/mixlunch-service-api" $HOME/.ghq/github.com/momotaro98/mixlunch-service-api
```

You'll see instructions like below.

```
/Users/shintaro/.ghq/github.com/momotaro98/mixlunch-service-api/partyservice/domain_test.go:8:2: import not sorted correctly. should be replace to
import (
        "errors"
        "testing"
        "time"

        "github.com/golang/mock/gomock"

        "github.com/momotaro98/mixlunch-service-api/userservice"
        "github.com/momotaro98/mixlunch-service-api/utils"
)
/Users/shintaro/.ghq/github.com/momotaro98/mixlunch-service-api/userservice/provider.go:5:2: import not sorted correctly. should be replace to
import (
        "github.com/google/wire"

        "github.com/momotaro98/mixlunch-service-api/tagservice"
)
/Users/shintaro/.ghq/github.com/momotaro98/mixlunch-service-api/wire.go:8:2: import not sorted correctly. should be replace to
import (
        "github.com/google/wire"

        "github.com/momotaro98/mixlunch-service-api/logger"
        "github.com/momotaro98/mixlunch-service-api/partyservice"
        "github.com/momotaro98/mixlunch-service-api/tagservice"
        usService "github.com/momotaro98/mixlunch-service-api/userscheduleservice"
        "github.com/momotaro98/mixlunch-service-api/userservice"
)
```

### Use as modifier

You can modify all of the target files by using `-w` option.

```shell script
$ strictgoimports -w -local "github.com/username/repo" .
```

This results in fixing import paths.
