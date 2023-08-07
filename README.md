# kache
Open source cloud-native web accelerator.

> :warning: Please note that the project is still in a very early stage of development and is not yet ready for production!

## Overview

kache is a modern cloud-native web accelerator and HTTP cache proxy that is highly available, reliable, and performant. It supports the latest RFC specifications and should be able to handle high traffic loads, be easily scalable, and support distributed caching systems. 

## Docs
For guidance on installation, development, deployment, and administration, see the [User Documentation](https://kacheio.github.io/docs/).

## Running kache

To run kache, get the latest binary from the [releases](https://github.com/kacheio/kache/releases) page and run it with the [sample configuration file](https://github.com/kacheio/kache/blob/main/kache.sample.yml):
```
./kache -config.file=kache.yml
```

Alternatively, use the official [Docker image](https://hub.docker.com/r/kacheio/kache) and run it with the sample configuration file:
```
docker run -it -p 80:80 -v $PWD/kache.yml:/etc/kache/kache.yml kache -config.file=/etc/kache/kache.yml 
````

Or, build from source:
```
git clone https://github.com/kacheio/kache
```

If you want to run kache with a distributed caching backend (e.g. Redis), you can use and run this example [docker-compose](https://github.com/kacheio/kache/blob/main/deploy/docker/docker-compose.yml) as a starting point:

```
docker-compose -f deploy/docker/docker-compose.yml up 
```

## Contributing 

We welcome contributions! If you're looking for issues to work on, we're happy to help. To get in touch, report bugs, suggest improvements, or request new features, help us by [opening an issue](https://github.com/kacheio/kache/issues/new). 

## License
kache is under the MIT License. See the [LICENSE](https://github.com/kacheio/kache/blob/main/LICENSE) file for details.

## Sponsors
 
kache is sponsored and supported by [Media Tech Lab](https://github.com/media-tech-lab).

<a href="https://www.media-lab.de/en/programs/media-tech-lab">
    <img src="https://raw.githubusercontent.com/media-tech-lab/.github/main/assets/mtl-powered-by.png" width="240" title="Media Tech Lab powered by logo">
</a>
