Looking at the errors, the Go application can't download dependencies (no internet access in sandbox), and after the previous fix attempt got dependencies working, it failed with wrong DB credentials. I need to:

1. Fix the Go source files to use the correct connection string
2. Set up go.mod/go.sum with vendored dependencies or use GONOSUMCHECK/GOFLAGS

The key issues are:
- No internet access for `go get` (need `GONOSUMCHECK` or vendor mode)
- Wrong DB credentials (need `postgres://app:app@db:5432/app?sslmode=disable`)

Let me create the necessary Go files and configure them properly.