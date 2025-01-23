# Shuffle CLI 
The Shuffle CLI helps you develop, test and deploy apps to Shuffle more easily.

**ALPHA: We are building this to check if it is of interest or not. It does not yet have a binary version published, so build it first: `go build`**

## Usage
```bash
$ shufflecli --help
```

## Apptesting
Since January 2025 you can test Shuffle Apps standalone outside Shuffle and Docker entirely. [See the App SDK details for more info](https://github.com/Shuffle/app_sdk/blob/main/README.md#usage).

**Static test an app**
```bash
$ shufflecli app test <filepath>
```

**Upload an app:**
```bash
$ shufflecli app upload <filepath>
```


## Coming features
- Binary releases: `GOOS=darwin GOARCH=arm64 go build -o shufflecli-macos-arm64`
- Testing scripts & functions by themselves
- Workflow building (maybe)
- Organization administration (maybe)
