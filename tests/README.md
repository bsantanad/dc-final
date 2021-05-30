# Tests for the Distributed and Parallel Image Processing

The following automation is going to be the one that will be used for
testing your final project. This is not a must but highly recommended
to test your system with this in order to make sure you have
everything in place for the final revision.

The provided step have been tested in the `cs-box` virtualbox.


## Video Utils

The [`video_utils.py`](./video_utils.py) scriipt is a helper for
extracting hundreds of images from a video. It will provide a high
volume of images that can be processed with your system.

- Install dependencies

```
sudo pacman -Sy python3 python-virtualenv python-pip --noconfirm
virtualenv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

- Download a sample video from [https://sample-videos.com/](https://sample-videos.com/)

```
curl -Ok https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4
```

- Run the  [`video_utils.py`](video_utils.py) script

```
python video_utils.py -action extract big_buck_bunny_720p_1mb.mp4 frames
```

The script will generate a new directory `frames` with ~130 images.



## Test suite

- Login into your system and save the generated `token`

```
curl -u username:password -X POST http://localhost:8080/login
```


- Get Sytem's status

```
curl -H "Authorization: Bearer <ACCESS_TOKEN>" http://localhost:8080/status
```


- Create new workload (save the `workload_id`)

```
curl -X POST -H "Authorization: Bearer <ACCESS_TOKEN>" http://localhost:8080/workloads
```


- Get details about workload

```
curl -H "Authorization: Bearer <ACCESS_TOKEN>" http://localhost:8080/workloads/<workload_id>
```



- Upload images

```
python stress_test.py -action push -workload-id <workload_id> -token <token> -frames-path frames
```


- Get details about workload

```
curl -H "Authorization: Bearer <ACCESS_TOKEN>" http://localhost:8080/workloads/<workload_id>
```


- Download filtered images

```
python3 stress_test.py -action pull -workload-id <workload_id> -image-type filtered -token <token> -frames-path filtered-frames
```


- Join filtered images into a new video

```
python3 video_utils.py -action join filtered.mp4 filtered
```


- Logout from your system
```
curl -X DELETE -H "Authorization: Bearer <ACCESS_TOKEN>" http://localhost:8080/logout
```
