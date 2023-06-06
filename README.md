# shepherd
A web gateway for serving Jackal files.

## Installing
```sh
make install
```

## Running with Docker

```shell
docker run -itd --publish 5656:5656 jackalmarston/shepherd:latest
```

## Using

```sh
shepherd
```

Visit `localhost:5656/f/{fid}` to view file directly or `localhost:5656/p/{owner_address}/{path_to_file}` to view the file through an abstracted path.
