# user guide

## installation

the projects use these non-standard packages
```bash
github.com/gorilla/mux
go.nanomsg.org/mangos
go get github.com/anthonynsimon/bild
```
you can install them by using
```bash
go get <package-name>
```
if you run into any problems, you can go into each project official page:

[mux](https://github.com/gorilla/mux)

[mangos](https://github.com/nanomsg/mangos)

[bild](https://github.com/anthonynsimon/bild)

you also need [grpc](https://grpc.io/docs/languages/go/quickstart/)

## initial setup

So running the whole system is quite simple, first lets look at the arch
![architecure](images/api-dist.png)

here you can see the communication protocols that are used across the system

here you can see the communication protocols that are used across the system

we focused on [orthogonality][def-ort] when designing this project. Meaning,
each component you see in the architecture can in fact run independently. You
could launch the API in a server in UK, then the controller in Mexico, the
scheduler in US and the system wouldn't know the difference.

But to make it easier to run in localhost, we start each component in
`main.go` (literally it just starts the other modules in the system go check it
out). So after cloning the project and `cd` into the directory go ahead and run
```bash
go run main.go
```
A message will come up :)

### workers

Next step is the workers. Workers can be set in any machine, again, the
system is orthogonal, to subscribe a worker just give it a name, and tell them
where your controller is, something like
```bash
go run worker/main.go --controller tcp://localhost:40901 --worker-name pedro
```
of course if you are running everything in different machines the controller
address won't be `localhost`, therefore you can do
```bash
go run worker/main.go \
    --controller <controller address> \
    --worker-name <any string will do>
```

You can repeat names don't worry. :) (each worker has a unique ID so no problem
on repeating names)

### uploading images

So cool, you now have your system with some workers there, what's next? Let's
upload some images!!

**These examples are in localhost please change port and
url accordingly to your system**

#### login

`/login` **POST**

First thing you will have to log in
```bash
curl -X POST -u user:password localhost:8080/login
```
This will return a token, use it in every request to the api, as an
bearer token.

#### create workload

`/workloads` **POST**

With our token in hand, lets create some workloads, these workloads are the
once that will take our images to the controller and tell them what filter to
apply

right now we support two --`blur` or `grayscale`-- filters, so you can do
something like this
```bash
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -X POST \
     -d '{"filter": "blur", "workload_name": "jose"}' \
     localhost:8080/workloads
```
or
```bash
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -X POST \
     -d '{"filter": "grayscale", "workload_name": "jose"}' \
     localhost:8080/workloads
```

_note:_ the workload name can be whatever


#### get info on workload

`/workloads/{workload_id}` **GET**
you can get all the info on a workload by using a GET

```bash
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -X GET \
     localhost:8080/workloads/{workload_id}
```

#### uploading images

`/images` **POST**

once you have your workers running, and have at least a workload
go ahead and upload your first image :)
```bash
curl -H "Content-Type: multipart/form-data" \
     -H "Authorization: Bearer am9zZTptYXJpYQ==" \
     -F "data=@<filename>" \
     -F "workload_id=<int workload_id>" \
     -F "type=original" \
     -X POST \
     localhost:8080/images
```

done! and image has been uploaded, you should get a json like this:
```bash
{
  "message": "An image has been successfully uploaded :)",
  "workload_id": 2,
  "image_id": 0,
  "type": "original",
  "size": 83888
}
```

#### see images

`/images` **GET**

okay so, when you upload an image, a lot of things happen internally,
but at the end, a **worker** will upload the filtered image to the API.

you can do this to get all the images in the api
```bash
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer am9zZTptYXJpYQ==" \
     -X GET \
     localhost:8080/images | jq
```
you will see something like this
```bash
[
  {
    "workload_id": 2,
    "image_id": 0,
    "type": "original",
    "size": 83888
  },
  {
    "workload_id": 0,
    "image_id": 1,
    "type": "filtered",
    "size": 188471
  }
]
```
as you can see, even though we just uploaded one image, now we have two in
the api. This means the worker has already work on it, and uploaded it to the
api

#### download images

`/images/{image_id}` **GET**

you can download images by id
```bash
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer am9zZTptYXJpYQ==" \
     -X GET \
     localhost:8080/images/1 \
     --output <filename>.png
```
#### check status

`/status` **GET**

finally you can check the overall status of the api by using, this will return
all workloads

```bash
curl -H "Content-Type: application/json" \
     -H "Authorization: Bearer am9zZTptYXJpYQ==" \
     -X GET \
     localhost:8080/status
```

#### other

`/status` **DELETE**

you can log out of the system using:

```bash
curl -X DELETE \
     -H "Authorization: Bearer am9zZTptYXJpYQ==" \
     localhost:8080/logout
```


### known problems

* first you need to create the workers and then upload images, if you try the
other way around it won't work. There's a fix for this, but it's still under
development, we are evaluating on using it, because it will take some of the
orthogonality of the project

* we broke DRY (don't repeat yourself), we are repeating the structs over
several files, nevertheless we are planning to make a perl script that
generates these structs automatically. We also repeat the workload fake
database this is because the project only works on runtime. If a proper
database is implemented there's no need to repeat this information


[def-ort]: https://flylib.com/books/en/1.315.1.23/1/


