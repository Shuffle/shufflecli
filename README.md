# Shuffle CLI 
The Shuffle CLI helps you develop, test and deploy apps to Shuffle more easily.

**ALPHA: We are building this to check if it is of interest or not. It does not yet have a binary version published, so build it first: `go build`**

## Usage
```bash
$ shufflecli --help
```

**Static test an app**
```bash
$ shufflecli app test <filepath>
```

**Upload an app:**
```bash
$ shufflecli app upload <filepath>
```


## Coming features
- Binary & versioned release(s)
- Testing scripts & functions by themselves
- Workflow building (maybe)
- Organization administration (maybe)
